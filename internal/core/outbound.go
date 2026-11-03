package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/service"
)

// Each protocol gets its own fields; the installed, pinned core is the final schema validator.
var protocolFields = map[string]string{
	"socks":       "version network udp_over_tcp",
	"http":        "tls path headers",
	"ssh":         "user private_key private_key_passphrase host_key host_key_algorithms client_version",
	"shadowsocks": "method plugin plugin_opts network udp_over_tcp multiplex",
	"vless":       "uuid flow network tls multiplex transport packet_encoding",
	"vmess":       "uuid security alter_id global_padding authenticated_length network tls packet_encoding multiplex transport",
	"trojan":      "network tls multiplex transport",
	"hysteria2":   "server_ports hop_interval up_mbps down_mbps obfs network tls brutal_debug",
	"tuic":        "uuid congestion_control udp_relay_mode udp_over_stream zero_rtt_handshake heartbeat network tls",
	"anytls":      "tls idle_session_check_interval idle_session_timeout min_idle_session",
}

func BuildOutbound(ep database.Endpoint) (map[string]any, error) {
	username, password, err := service.DynamicCredentials(ep)
	if err != nil {
		return nil, err
	}
	ep.Username, ep.Password = username, password
	o := map[string]any{}
	if ep.Options != "" {
		if err := json.Unmarshal([]byte(ep.Options), &o); err != nil || o == nil {
			return nil, fmt.Errorf("%s: Options 必须为 JSON 对象", ep.Tag)
		}
	}
	typ := ep.Type
	if typ == "custom" {
		var ok bool
		typ, ok = o["type"].(string)
		if !ok {
			return nil, fmt.Errorf("自定义出站缺少 type")
		}
		// Tags, dependencies and credentials still use the managed node model.
		if _, ok := o["detour"]; ok {
			return nil, fmt.Errorf("请使用节点的 Detour 字段")
		}
	} else {
		fields, ok := protocolFields[typ]
		if !ok {
			return nil, fmt.Errorf("不支持的出站协议: %s", typ)
		}
		for k := range o {
			if !strings.Contains(" "+fields+" connect_timeout tcp_fast_open tcp_multi_path udp_fragment domain_resolver ", " "+k+" ") {
				return nil, fmt.Errorf("%s 不支持字段 %s", typ, k)
			}
		}
	}
	if _, ok := protocolFields[typ]; !ok {
		return nil, fmt.Errorf("自定义出站类型 %s 尚不支持管理", typ)
	}
	o["type"], o["tag"] = typ, ep.Tag
	if ep.Server != "" {
		o["server"] = ep.Server
	}
	if ep.ServerPort != 0 {
		o["server_port"] = ep.ServerPort
	}
	server, _ := o["server"].(string)
	if strings.TrimSpace(server) == "" || strings.ContainsAny(server, "/ @\t\r\n") {
		return nil, fmt.Errorf("代理地址无效")
	}
	port := ep.ServerPort
	if port == 0 {
		if n, ok := o["server_port"].(float64); ok {
			port = int(n)
		}
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("端口必须在 1–65535")
	}
	if ep.Password != "" && typ != "vless" && typ != "vmess" {
		o["password"] = ep.Password
	}
	if typ == "socks" || typ == "http" {
		if ep.Username != "" {
			o["username"] = ep.Username
		}
	}
	if typ == "ssh" && ep.Username != "" {
		o["user"] = ep.Username
	}
	if typ == "socks" {
		if _, ok := o["version"]; !ok {
			o["version"] = "5"
		}
		if v := o["version"]; v != "4" && v != "4a" && v != "5" {
			return nil, fmt.Errorf("SOCKS version 必须是 4 / 4a / 5")
		}
		if o["version"] != "5" {
			if ep.Password != "" {
				return nil, fmt.Errorf("SOCKS4/4a 不支持密码")
			}
			o["network"] = "tcp"
		}
	}
	required := map[string][]string{"ssh": {"user"}, "shadowsocks": {"method", "password"}, "vless": {"uuid"}, "vmess": {"uuid", "security"}, "trojan": {"password"}, "hysteria2": {"password", "tls"}, "tuic": {"uuid", "password", "tls"}, "anytls": {"password", "tls"}}
	for _, k := range required[typ] {
		if o[k] == nil || o[k] == "" {
			return nil, fmt.Errorf("%s 缺少 %s", typ, k)
		}
	}
	if ep.Detour != "" {
		o["detour"] = ep.Detour
	}
	return o, nil
}

// ValidateGraph rejects cycles and missing dependencies, including disabled nodes.
func ValidateGraph(nodes []database.Endpoint) error {
	byTag := map[string]database.Endpoint{}
	for _, n := range nodes {
		if n.Tag == "direct" || n.Tag == "block" || n.Tag == "" {
			return fmt.Errorf("保留或空标签")
		}
		if _, ok := byTag[n.Tag]; ok {
			return fmt.Errorf("重复标签 %s", n.Tag)
		}
		byTag[n.Tag] = n
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(tag string) error {
		if state[tag] == 1 {
			return fmt.Errorf("中继循环: %s", tag)
		}
		if state[tag] == 2 {
			return nil
		}
		state[tag] = 1
		n := byTag[tag]
		if n.Detour != "" {
			if _, ok := byTag[n.Detour]; !ok {
				return fmt.Errorf("%s 缺少中继 %s", tag, n.Detour)
			}
			if err := visit(n.Detour); err != nil {
				return err
			}
		}
		state[tag] = 2
		return nil
	}
	for tag := range byTag {
		if err := visit(tag); err != nil {
			return err
		}
	}
	return nil
}

func NodeAvailable(tag string, nodes []database.Endpoint) bool {
	seen := map[string]bool{}
	for tag != "" {
		if seen[tag] {
			return false
		}
		seen[tag] = true
		found := false
		for _, n := range nodes {
			if n.Tag == tag {
				if !n.Enable {
					return false
				}
				tag = n.Detour
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// NodeConfig is shared by export, validation and isolated real network probes.
func NodeConfig(nodes []database.Endpoint, target string) (map[string]any, error) {
	if err := ValidateGraph(nodes); err != nil {
		return nil, err
	}
	obs := []any{}
	eps := []any{}
	included := map[string]bool{}
	var add func(string) error
	add = func(tag string) error {
		if included[tag] {
			return nil
		}
		included[tag] = true
		for _, n := range nodes {
			if n.Tag == tag {
				if !NodeAvailable(tag, nodes) {
					return fmt.Errorf("节点或中继已禁用: %s", tag)
				}
				if n.Detour != "" {
					if err := add(n.Detour); err != nil {
						return err
					}
				}
				if n.Type == "wireguard" {
					e, err := buildEndpoint(n)
					if err != nil {
						return err
					}
					eps = append(eps, e)
				} else {
					o, err := BuildOutbound(n)
					if err != nil {
						return err
					}
					obs = append(obs, o)
				}
				return nil
			}
		}
		return fmt.Errorf("节点不存在: %s", tag)
	}
	if err := add(target); err != nil {
		return nil, err
	}
	return map[string]any{"outbounds": obs, "endpoints": eps, "route": map[string]any{"final": target}}, nil
}

func CheckConfig(cfg any) error {
	f, err := os.CreateTemp("", "slinx-check-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = json.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		return err
	}
	f.Close()
	out, err := exec.Command(Default.BinPath, "check", "-c", f.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sing-box 配置校验失败: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

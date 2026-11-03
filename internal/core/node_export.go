package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/slinxlink/node/internal/database"
)

type NodeBundle struct {
	Format string              `json:"format"`
	Nodes  []database.Endpoint `json:"nodes"`
}

func ExportNodes(nodes []database.Endpoint, tags []string, format string) (string, error) {
	selected := []database.Endpoint{}
	seen := map[string]bool{}
	var add func(string) error
	add = func(tag string) error {
		if seen[tag] {
			return nil
		}
		seen[tag] = true
		for _, n := range nodes {
			if n.Tag == tag {
				if n.Detour != "" {
					if err := add(n.Detour); err != nil {
						return err
					}
				}
				selected = append(selected, n)
				return nil
			}
		}
		return fmt.Errorf("缺少节点 %s", tag)
	}
	if err := ValidateGraph(nodes); err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("请选择节点")
	}
	for _, t := range tags {
		if err := add(t); err != nil {
			return "", err
		}
	}
	encode := func(v any) (string, error) { b, err := json.MarshalIndent(v, "", "  "); return string(b), err }
	if format == "slinx-ef" {
		return encode(NodeBundle{Format: "slinx-ef/nodes-v1", Nodes: selected})
	}
	obs := []any{}
	eps := []any{}
	for _, n := range selected {
		if n.Type == "wireguard" {
			e, err := buildEndpoint(n)
			if err != nil {
				return "", err
			}
			eps = append(eps, e)
		} else {
			o, err := BuildOutbound(n)
			if err != nil {
				return "", err
			}
			obs = append(obs, o)
		}
	}
	if format == "sing-box" {
		cfg := map[string]any{"outbounds": obs, "endpoints": eps, "route": map[string]any{"final": tags[0]}}
		if err := CheckConfig(cfg); err != nil {
			return "", err
		}
		return encode(cfg)
	}
	if len(eps) > 0 {
		return "", fmt.Errorf("%s 的 WireGuard 转换尚不支持，请导出 sing-box", format)
	}
	proxies := []any{}
	lines := []string{}
	for _, raw := range obs {
		o := raw.(map[string]any)
		switch format {
		case "mihomo":
			p, err := mihomoNode(o)
			if err != nil {
				return "", err
			}
			proxies = append(proxies, p)
		case "surge":
			p, err := surgeNode(o)
			if err != nil {
				return "", err
			}
			lines = append(lines, p)
		case "uri":
			p, err := nodeURI(o)
			if err != nil {
				return "", err
			}
			lines = append(lines, p)
		default:
			return "", fmt.Errorf("不支持的导出格式")
		}
	}
	switch format {
	case "mihomo":
		b, err := yaml.Marshal(map[string]any{"proxies": proxies, "rules": []string{"MATCH," + tags[0]}})
		return string(b), err
	case "surge":
		return "[Proxy]\n" + strings.Join(lines, "\n") + "\n[Rule]\nFINAL," + tags[0] + "\n", nil
	default:
		return strings.Join(lines, "\n"), nil
	}
}

func rejectFields(o map[string]any, allowed string) error {
	for k := range o {
		if !strings.Contains(" "+allowed+" ", " "+k+" ") {
			return fmt.Errorf("目标格式不支持 %s 的字段 %s，请使用 sing-box", o["type"], k)
		}
	}
	return nil
}

func mihomoNode(o map[string]any) (map[string]any, error) {
	if err := rejectFields(o, "type tag server server_port username password version uuid security alter_id method flow tls transport network detour congestion_control udp_relay_mode up_mbps down_mbps obfs packet_encoding"); err != nil {
		return nil, err
	}
	typ := fmt.Sprint(o["type"])
	switch typ {
	case "socks":
		if o["version"] != "5" {
			return nil, fmt.Errorf("Mihomo 不支持 SOCKS4/4a")
		}
		typ = "socks5"
	case "shadowsocks":
		typ = "ss"
	case "http", "vless", "vmess", "trojan", "hysteria2", "tuic", "anytls":
	default:
		return nil, fmt.Errorf("Mihomo 转换尚不支持 %s", typ)
	}
	p := map[string]any{"name": o["tag"], "type": typ, "server": o["server"], "port": o["server_port"]}
	keys := map[string]string{"username": "username", "password": "password", "uuid": "uuid", "security": "cipher", "method": "cipher", "alter_id": "alterId", "flow": "flow", "detour": "dialer-proxy", "congestion_control": "congestion-controller", "udp_relay_mode": "udp-relay-mode", "packet_encoding": "packet-encoding", "up_mbps": "up", "down_mbps": "down"}
	for a, b := range keys {
		if v, ok := o[a]; ok {
			p[b] = v
		}
	}
	if typ != "http" {
		p["udp"] = o["network"] != "tcp"
	}
	if t, ok := o["tls"].(map[string]any); ok {
		if err := rejectFields(t, "enabled server_name insecure alpn utls reality"); err != nil {
			return nil, err
		}
		p["tls"] = t["enabled"]
		if v, ok := t["server_name"]; ok {
			key := "servername"
			if typ == "hysteria2" || typ == "tuic" || typ == "anytls" || typ == "http" {
				key = "sni"
			}
			p[key] = v
		}
		if v, ok := t["insecure"]; ok {
			p["skip-cert-verify"] = v
		}
		if v, ok := t["alpn"]; ok {
			p["alpn"] = v
		}
		if u, ok := t["utls"].(map[string]any); ok {
			if err := rejectFields(u, "enabled fingerprint"); err != nil {
				return nil, err
			}
			p["client-fingerprint"] = u["fingerprint"]
		}
		if r, ok := t["reality"].(map[string]any); ok {
			if err := rejectFields(r, "enabled public_key short_id"); err != nil {
				return nil, err
			}
			p["reality-opts"] = map[string]any{"public-key": r["public_key"], "short-id": r["short_id"]}
		}
	}
	if t, ok := o["transport"].(map[string]any); ok {
		if err := rejectFields(t, "type path headers service_name"); err != nil {
			return nil, err
		}
		switch t["type"] {
		case "ws":
			p["network"] = "ws"
			p["ws-opts"] = map[string]any{"path": t["path"], "headers": t["headers"]}
		case "grpc":
			p["network"] = "grpc"
			p["grpc-opts"] = map[string]any{"grpc-service-name": t["service_name"]}
		default:
			return nil, fmt.Errorf("Mihomo 不支持此传输层转换")
		}
	}
	if obfs, ok := o["obfs"].(map[string]any); ok {
		if obfs["type"] != "salamander" {
			return nil, fmt.Errorf("不支持的混淆转换")
		}
		p["obfs"] = "salamander"
		p["obfs-password"] = obfs["password"]
	}
	return p, nil
}

func surgeNode(o map[string]any) (string, error) {
	if err := rejectFields(o, "type tag server server_port username password version method tls detour"); err != nil {
		return "", err
	}
	typ := fmt.Sprint(o["type"])
	switch typ {
	case "socks":
		if o["version"] != "5" {
			return "", fmt.Errorf("Surge 转换不支持 SOCKS4/4a")
		}
		typ = "socks5"
	case "http", "trojan":
	case "shadowsocks":
		typ = "ss"
	default:
		return "", fmt.Errorf("Surge 转换不支持 %s", typ)
	}
	for _, k := range []string{"tag", "server", "username", "password", "detour", "method"} {
		if strings.ContainsAny(fmt.Sprint(o[k]), ",\r\n=\"") {
			return "", fmt.Errorf("Surge 字段 %s 包含不能安全转换的字符", k)
		}
	}
	parts := []string{fmt.Sprint(o["tag"]) + " = " + typ, fmt.Sprint(o["server"]), fmt.Sprint(o["server_port"])}
	for _, k := range []string{"username", "password", "detour", "method"} {
		if v, ok := o[k]; ok {
			key := k
			if k == "detour" {
				key = "underlying-proxy"
			}
			if k == "method" {
				key = "encrypt-method"
			}
			parts = append(parts, key+"="+fmt.Sprint(v))
		}
	}
	if t, ok := o["tls"].(map[string]any); ok {
		if err := rejectFields(t, "enabled server_name insecure"); err != nil {
			return "", err
		}
		if t["enabled"] == true {
			parts = append(parts, "tls=true")
		}
		if v, ok := t["server_name"]; ok {
			if strings.ContainsAny(fmt.Sprint(v), ",\r\n") {
				return "", fmt.Errorf("SNI 无法转换")
			}
			parts = append(parts, "sni="+fmt.Sprint(v))
		}
		if t["insecure"] == true {
			parts = append(parts, "skip-cert-verify=true")
		}
	}
	return strings.Join(parts, ", "), nil
}

func nodeURI(o map[string]any) (string, error) {
	if o["detour"] != nil {
		return "", fmt.Errorf("URI 不支持中继依赖，请使用 sing-box")
	}
	if err := rejectFields(o, "type tag server server_port username password version method uuid flow security alter_id tls transport congestion_control udp_relay_mode up_mbps down_mbps obfs network"); err != nil {
		return "", err
	}
	typ := fmt.Sprint(o["type"])
	q := url.Values{}
	user := url.User(fmt.Sprint(o["password"]))
	switch typ {
	case "vless":
		user = url.User(fmt.Sprint(o["uuid"]))
		q.Set("encryption", "none")
		if v, ok := o["flow"]; ok {
			q.Set("flow", fmt.Sprint(v))
		}
	case "trojan", "hysteria2", "anytls":
	case "tuic":
		user = url.UserPassword(fmt.Sprint(o["uuid"]), fmt.Sprint(o["password"]))
	case "socks", "http":
		name, _ := o["username"].(string)
		pass, _ := o["password"].(string)
		user = url.UserPassword(name, pass)
		if typ == "socks" {
			if o["version"] != "5" {
				return "", fmt.Errorf("URI 转换仅支持 SOCKS5")
			}
			typ = "socks5"
		}
	case "shadowsocks":
		typ = "ss"
		user = url.User(base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprint(o["method"]) + ":" + fmt.Sprint(o["password"]))))
	case "vmess":
		// VMess uses the established v2rayN JSON link schema.
		v := map[string]any{"v": "2", "ps": o["tag"], "add": o["server"], "port": fmt.Sprint(o["server_port"]), "id": o["uuid"], "aid": "0", "scy": o["security"], "net": "tcp", "type": "none"}
		if aid, ok := o["alter_id"]; ok {
			v["aid"] = fmt.Sprint(aid)
		}
		if t, ok := o["tls"].(map[string]any); ok {
			if err := rejectFields(t, "enabled server_name insecure alpn"); err != nil {
				return "", err
			}
			if t["enabled"] == true {
				v["tls"] = "tls"
			}
			v["sni"] = t["server_name"]
			v["allowInsecure"] = t["insecure"]
		}
		if t, ok := o["transport"].(map[string]any); ok {
			if t["type"] != "ws" {
				return "", fmt.Errorf("VMess URI 仅支持 TCP/WS")
			}
			v["net"] = "ws"
			v["path"] = t["path"]
			if h, ok := t["headers"].(map[string]any); ok {
				v["host"] = h["Host"]
			}
		}
		b, _ := json.Marshal(v)
		return "vmess://" + base64.StdEncoding.EncodeToString(b), nil
	default:
		return "", fmt.Errorf("URI 转换不支持 %s", typ)
	}
	if t, ok := o["tls"].(map[string]any); ok {
		if err := rejectFields(t, "enabled server_name insecure alpn utls reality"); err != nil {
			return "", err
		}
		if t["enabled"] == true {
			q.Set("security", "tls")
		}
		if s, ok := t["server_name"].(string); ok {
			q.Set("sni", s)
		}
		if t["insecure"] == true {
			q.Set("insecure", "1")
		}
		if a, ok := t["alpn"].([]any); ok {
			parts := []string{}
			for _, x := range a {
				parts = append(parts, fmt.Sprint(x))
			}
			q.Set("alpn", strings.Join(parts, ","))
		}
		if u, ok := t["utls"].(map[string]any); ok {
			q.Set("fp", fmt.Sprint(u["fingerprint"]))
		}
		if r, ok := t["reality"].(map[string]any); ok {
			q.Set("security", "reality")
			q.Set("pbk", fmt.Sprint(r["public_key"]))
			q.Set("sid", fmt.Sprint(r["short_id"]))
		}
		if typ == "http" && t["enabled"] == true {
			typ = "https"
		}
	}
	if t, ok := o["transport"].(map[string]any); ok {
		if err := rejectFields(t, "type path headers service_name"); err != nil {
			return "", err
		}
		q.Set("type", fmt.Sprint(t["type"]))
		if v, ok := t["path"]; ok {
			q.Set("path", fmt.Sprint(v))
		}
		if v, ok := t["service_name"]; ok {
			q.Set("serviceName", fmt.Sprint(v))
		}
		if h, ok := t["headers"].(map[string]any); ok {
			for k, v := range h {
				if k != "Host" {
					return "", fmt.Errorf("URI 不支持传输层 header %s", k)
				}
				q.Set("host", fmt.Sprint(v))
			}
		}
	}
	for _, k := range []string{"congestion_control", "udp_relay_mode", "up_mbps", "down_mbps", "obfs", "network"} {
		if _, ok := o[k]; ok {
			return "", fmt.Errorf("URI 无法无损转换 %s，请使用 sing-box", k)
		}
	}
	host := net.JoinHostPort(fmt.Sprint(o["server"]), fmt.Sprint(o["server_port"]))
	u := url.URL{Scheme: typ, Host: host, User: user, RawQuery: q.Encode(), Fragment: fmt.Sprint(o["tag"])}
	return u.String(), nil
}

func ImportNodes(content string) ([]database.Endpoint, error) {
	var bundle NodeBundle
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&bundle); err == nil && bundle.Format == "slinx-ef/nodes-v1" {
		return bundle.Nodes, nil
	}
	var cfg struct {
		Outbounds []map[string]any `json:"outbounds"`
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(content), &cfg); err == nil && (len(cfg.Outbounds) > 0 || len(cfg.Endpoints) > 0) {
		var top map[string]any
		json.Unmarshal([]byte(content), &top)
		for k := range top {
			if k != "outbounds" && k != "endpoints" && k != "route" {
				return nil, fmt.Errorf("节点导入不支持顶层字段 %s", k)
			}
		}
		result := []database.Endpoint{}
		for _, o := range append(cfg.Outbounds, cfg.Endpoints...) {
			n, err := importOutbound(o)
			if err != nil {
				return nil, err
			}
			result = append(result, n)
		}
		return result, nil
	}
	result := []database.Endpoint{}
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		o, err := parseNodeURI(strings.TrimSpace(line))
		if err != nil {
			return nil, err
		}
		n, err := importOutbound(o)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("未找到可导入节点")
	}
	return result, nil
}

func importOutbound(o map[string]any) (database.Endpoint, error) {
	n := database.Endpoint{Enable: true}
	n.Type, _ = o["type"].(string)
	n.Tag, _ = o["tag"].(string)
	n.Detour, _ = o["detour"].(string)
	if n.Type == "wireguard" {
		if err := rejectFields(o, "type tag detour mtu address private_key peers udp_timeout workers"); err != nil {
			return n, err
		}
		b, _ := json.Marshal(o)
		var e endpoints
		if err := json.Unmarshal(b, &e); err != nil {
			return n, err
		}
		if len(e.Peers) != 1 {
			return n, fmt.Errorf("WireGuard 导入需要单 peer")
		}
		n.MTU = e.MTU
		n.PrivateKey = e.PrivateKey
		n.Workers = e.Workers
		n.PeerAddress = e.Peers[0].Address
		n.PeerPort = e.Peers[0].Port
		n.PeerPublicKey = e.Peers[0].PublicKey
		n.AllowedIPs = strings.Join(e.Peers[0].AllowedIPs, ",")
		a, _ := json.Marshal(e.Address)
		n.Address = string(a)
		r, _ := json.Marshal(e.Peers[0].Reserved)
		n.Reserved = string(r)
		if e.UDPTimeout != "" {
			return n, fmt.Errorf("WireGuard udp_timeout 导入尚不支持，请使用 SLINX-EF 格式")
		}
		return n, nil
	}
	n.Server, _ = o["server"].(string)
	if v, ok := o["server_port"].(float64); ok {
		n.ServerPort = int(v)
	} else if v, ok := o["server_port"].(int); ok {
		n.ServerPort = v
	}
	n.Username, _ = o["username"].(string)
	n.Password, _ = o["password"].(string)
	for _, k := range []string{"type", "tag", "detour", "server", "server_port", "username", "password"} {
		delete(o, k)
	}
	b, _ := json.Marshal(o)
	n.Options = string(b)
	_, err := BuildOutbound(n)
	return n, err
}

func parseNodeURI(line string) (map[string]any, error) {
	u, err := url.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("URI 格式错误")
	}
	if u.Scheme == "vmess" {
		return parseVMessURI(strings.TrimPrefix(line, "vmess://"))
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, fmt.Errorf("URI 缺少有效端口")
	}
	typ := u.Scheme
	if typ == "hy2" {
		typ = "hysteria2"
	}
	if typ == "socks5" {
		typ = "socks"
	}
	if typ == "https" {
		typ = "http"
	}
	if typ == "ss" {
		typ = "shadowsocks"
	}
	tag := u.Fragment
	if tag == "" {
		tag = typ + "-" + u.Hostname() + "-" + u.Port()
	}
	o := map[string]any{"type": typ, "tag": tag, "server": u.Hostname(), "server_port": port}
	q := u.Query()
	for k := range q {
		if !strings.Contains(" security sni peer insecure allowInsecure alpn fp pbk sid flow encryption type path host serviceName ", " "+k+" ") {
			return nil, fmt.Errorf("URI 不支持参数 %s", k)
		}
	}
	if u.User == nil {
		return nil, fmt.Errorf("URI 缺少认证字段")
	}
	switch typ {
	case "vless":
		o["uuid"] = u.User.Username()
		if q.Get("flow") != "" {
			o["flow"] = q.Get("flow")
		}
	case "trojan", "hysteria2", "anytls":
		o["password"] = u.User.Username()
	case "tuic":
		o["uuid"] = u.User.Username()
		o["password"], _ = u.User.Password()
	case "socks", "http":
		o["username"] = u.User.Username()
		o["password"], _ = u.User.Password()
		if typ == "socks" {
			o["version"] = "5"
		}
	case "shadowsocks":
		raw := u.User.Username()
		if pass, ok := u.User.Password(); ok {
			raw += ":" + pass
		} else {
			b, e := decodeBase64(raw)
			if e != nil {
				return nil, e
			}
			raw = string(b)
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("SS URI 认证无效")
		}
		o["method"], o["password"] = parts[0], parts[1]
	default:
		return nil, fmt.Errorf("URI 协议不支持: %s", typ)
	}
	if q.Get("encryption") != "" && q.Get("encryption") != "none" {
		return nil, fmt.Errorf("不支持此 VLESS encryption")
	}
	tlsEnabled := q.Get("security") == "tls" || q.Get("security") == "reality" || u.Scheme == "https" || typ == "trojan" || typ == "hysteria2" || typ == "tuic" || typ == "anytls"
	if tlsEnabled {
		t := map[string]any{"enabled": true}
		sni := q.Get("sni")
		if sni == "" {
			sni = q.Get("peer")
		}
		if sni != "" {
			t["server_name"] = sni
		}
		if q.Get("insecure") == "1" || q.Get("allowInsecure") == "1" {
			t["insecure"] = true
		}
		if q.Get("alpn") != "" {
			t["alpn"] = strings.Split(q.Get("alpn"), ",")
		}
		if q.Get("fp") != "" {
			t["utls"] = map[string]any{"enabled": true, "fingerprint": q.Get("fp")}
		}
		if q.Get("security") == "reality" {
			t["reality"] = map[string]any{"enabled": true, "public_key": q.Get("pbk"), "short_id": q.Get("sid")}
		}
		o["tls"] = t
	}
	if netw := q.Get("type"); netw != "" && netw != "tcp" {
		t := map[string]any{"type": netw}
		switch netw {
		case "ws":
			t["path"] = q.Get("path")
			if q.Get("host") != "" {
				t["headers"] = map[string]any{"Host": q.Get("host")}
			}
		case "grpc":
			t["service_name"] = q.Get("serviceName")
		default:
			return nil, fmt.Errorf("URI 不支持传输层 %s", netw)
		}
		o["transport"] = t
	}
	return o, nil
}

func decodeBase64(s string) ([]byte, error) {
	for _, e := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := e.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("Base64 无效")
}

func parseVMessURI(raw string) (map[string]any, error) {
	b, err := decodeBase64(raw)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return nil, fmt.Errorf("VMess URI JSON 无效")
	}
	if err := rejectFields(v, "v ps add port id aid scy net type host path tls sni alpn fp allowInsecure"); err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(fmt.Sprint(v["port"]))
	aid, _ := strconv.Atoi(fmt.Sprint(v["aid"]))
	sec := v["scy"]
	if sec == nil || sec == "" {
		sec = "auto"
	}
	tag, _ := v["ps"].(string)
	if tag == "" {
		tag = "vmess-" + fmt.Sprint(v["add"])
	}
	o := map[string]any{"type": "vmess", "tag": tag, "server": v["add"], "server_port": port, "uuid": v["id"], "security": sec, "alter_id": aid}
	if v["tls"] == "tls" {
		t := map[string]any{"enabled": true}
		if s, ok := v["sni"].(string); ok && s != "" {
			t["server_name"] = s
		}
		if v["allowInsecure"] == true || v["allowInsecure"] == "1" {
			t["insecure"] = true
		}
		o["tls"] = t
	}
	if v["net"] == "ws" {
		o["transport"] = map[string]any{"type": "ws", "path": v["path"], "headers": map[string]any{"Host": v["host"]}}
	} else if v["net"] != nil && v["net"] != "tcp" && v["net"] != "" {
		return nil, fmt.Errorf("VMess URI 只支持 TCP/WS")
	}
	return o, nil
}

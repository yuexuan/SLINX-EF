package core

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/slinxlink/node/internal/database"
)

func protocolFixtures() []database.Endpoint {
	uuid := `"uuid":"11111111-1111-4111-8111-111111111111"`
	tls := `"tls":{"enabled":true,"server_name":"example.com"}`
	nodes := []database.Endpoint{}
	for _, v := range []struct{ typ, opt string }{
		{"socks", `{"version":"4"}`}, {"socks", `{"version":"4a"}`}, {"socks", `{"version":"5"}`},
		{"http", `{}`}, {"http", `{` + tls + `}`}, {"ssh", `{"user":"test","host_key":[]}`},
		{"shadowsocks", `{"method":"aes-128-gcm"}`}, {"vless", `{` + uuid + `,` + tls + `}`},
		{"vmess", `{` + uuid + `,"security":"auto","transport":{"type":"ws","path":"/ws"}}`},
		{"trojan", `{` + tls + `}`}, {"hysteria2", `{` + tls + `,"obfs":{"type":"salamander","password":"test"},"up_mbps":20,"down_mbps":50}`},
		{"tuic", `{` + uuid + `,` + tls + `,"congestion_control":"bbr","udp_relay_mode":"native"}`},
		{"anytls", `{` + tls + `,"idle_session_timeout":"30s"}`}, {"custom", `{"type":"socks","version":"5"}`},
	} {
		ep := database.Endpoint{Enable: true, Tag: v.typ, Type: v.typ, Server: "127.0.0.1", ServerPort: 12345, Options: v.opt}
		if !strings.Contains(v.opt, `"version":"4`) {
			ep.Password = "test-password"
		}
		nodes = append(nodes, ep)
	}
	return nodes
}

func TestNativeProtocolMatrix(t *testing.T) {
	previous := Default.BinPath
	defer func() { Default.BinPath = previous }()
	corePath := os.Getenv("EF_CORE_PATH")
	if corePath != "" {
		Default.BinPath = corePath
	}
	for i, ep := range protocolFixtures() {
		t.Run(ep.Type+string(rune('A'+i)), func(t *testing.T) {
			o, err := BuildOutbound(ep)
			if err != nil {
				t.Fatal(err)
			}
			if ep.Type == "ssh" {
				if o["user"] != "test" || o["username"] != nil {
					t.Fatal("SSH auth mapping")
				}
			}
			if corePath != "" {
				cfg, err := NodeConfig([]database.Endpoint{ep}, ep.Tag)
				if err != nil {
					t.Fatal(err)
				}
				if err = CheckConfig(cfg); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestNodeGraphAndExport(t *testing.T) {
	a := database.Endpoint{Enable: true, Tag: "relay", Type: "socks", Server: "127.0.0.1", ServerPort: 1080}
	b := a
	b.Tag = "exit"
	b.Detour = "relay"
	nodes := []database.Endpoint{a, b}
	if err := ValidateGraph(nodes); err != nil {
		t.Fatal(err)
	}
	nodes[0].Detour = "exit"
	if ValidateGraph(nodes) == nil {
		t.Fatal("cycle accepted")
	}
	nodes[0].Detour = ""
	nodes[0].Enable = false
	if NodeAvailable("exit", nodes) {
		t.Fatal("disabled relay accepted")
	}
	nodes[0].Enable = true
	data, err := ExportNodes(nodes, []string{"exit"}, "slinx-ef")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ImportNodes(data)
	if err != nil || len(got) != 2 || got[1].Detour != "relay" {
		t.Fatalf("chain roundtrip: %v", err)
	}
	if _, err := ExportNodes(nodes, []string{"exit"}, "uri"); err == nil {
		t.Fatal("URI lost dependency")
	}
	if _, err := BuildOutbound(database.Endpoint{Tag: "invalid", Type: "http", Server: "example.com", ServerPort: 80, Options: `{"uuid":"invalid"}`}); err == nil {
		t.Fatal("unknown field accepted")
	}
	for _, ep := range protocolFixtures() {
		if ep.Type == "ssh" || ep.Type == "custom" {
			continue
		}
		if ep.Type == "hysteria2" || ep.Type == "tuic" {
			continue
		} // Extended UDP parameters are intentionally rejected by URI conversion.
		if strings.Contains(ep.Options, `"version":"4`) {
			continue
		}
		o, err := BuildOutbound(ep)
		if err != nil {
			t.Fatal(err)
		}
		uri, err := nodeURI(o)
		if err != nil {
			t.Fatal(err)
		}
		round, err := ImportNodes(uri)
		if err != nil || len(round) != 1 {
			t.Fatalf("%s URI roundtrip: %v", ep.Type, err)
		}
		original, _ := json.Marshal(o)
		rebuilt, _ := BuildOutbound(round[0])
		actual, _ := json.Marshal(rebuilt)
		if !strings.Contains(string(actual), ep.Server) {
			t.Fatalf("URI dropped server: %s -> %s", original, actual)
		}
	}
}

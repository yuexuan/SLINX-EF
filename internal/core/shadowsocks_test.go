package core

import (
	"testing"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func TestBuildShadowsocksInbound(t *testing.T) {
	for _, method := range []string{"aes-128-gcm", "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm"} {
		serverKey, err := util.GenerateShadowsocksPassword(method)
		if err != nil {
			t.Fatal(err)
		}
		inbound := database.Inbound{Protocol: "shadowsocks", Port: 8388, ShadowsocksMethod: method, ShadowsocksPassword: serverKey}
		pass := util.ShadowsocksUserPassword(method, serverKey, "token-123")
		result, err := buildInbound(inbound, []user{{Name: "alice", Password: pass}})
		if err != nil {
			t.Fatal(err)
		}
		if result.Type != "shadowsocks" || result.Method != method || result.Password != serverKey {
			t.Fatalf("invalid core Shadowsocks configuration: %+v", result)
		}
		if len(result.Users) != 1 || result.Users[0].Password != pass || result.Users[0].UUID != "" {
			t.Fatalf("user credentials mismatch: %+v", result.Users)
		}
		if _, err := buildInbound(inbound, nil); err == nil {
			t.Fatal("empty multi-user inbound must not open single-key listener")
		}
	}
}

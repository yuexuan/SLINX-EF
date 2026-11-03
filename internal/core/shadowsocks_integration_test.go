package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func TestShadowsocksActualSingBoxCheck(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("bundled Windows sing-box core only")
	}
	binary := filepath.Join("..", "..", ".ci", "full-core-gvisor", "sing-box.exe")
	if _, err := os.Stat(binary); err != nil {
		t.Skip("pinned sing-box core not available")
	}

	for _, method := range []string{"aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305", "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm"} {
		t.Run(method, func(t *testing.T) {
			serverKey, err := util.GenerateShadowsocksPassword(method)
			if err != nil {
				t.Fatal(err)
			}
			ib := database.Inbound{Protocol: "shadowsocks", Port: 38388, ShadowsocksMethod: method, ShadowsocksPassword: serverKey}
			password := util.ShadowsocksUserPassword(method, serverKey, "user-token")
			inbound, err := buildShadowsocks(ib, []user{{Name: "test", Password: password}})
			if err != nil {
				t.Fatal(err)
			}
			cfg := map[string]any{
				"log":       map[string]any{"level": "error"},
				"inbounds":  []inbounds{inbound},
				"outbounds": []map[string]any{{"type": "direct", "tag": "direct"}},
				"route":     map[string]any{"final": "direct"},
			}
			raw, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "ss.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "check", "-c", path)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("sing-box rejected %s: %v\n%s", method, err, string(output))
			}
		})
	}
}

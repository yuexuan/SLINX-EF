package sub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func TestShadowsocksExports(t *testing.T) {
	user := database.User{Name: "alice", Token: "test-token", Password: "different-trojan-password"}
	for _, tc := range []struct {
		method string
		legacy bool
		surge  bool
	}{
		{"aes-128-gcm", true, true},
		{"2022-blake3-aes-128-gcm", false, true},
		{"2022-blake3-aes-256-gcm", false, true},
	} {
		serverKey, err := util.GenerateShadowsocksPassword(tc.method)
		if err != nil {
			t.Fatal(err)
		}
		ib := database.Inbound{Name: "SS-test", Protocol: "shadowsocks", Port: 8388, ShadowsocksMethod: tc.method, ShadowsocksPassword: serverKey}
		want := util.ShadowsocksClientPassword(tc.method, serverKey, user.Token)
		uri := shadowsocksURI(user, "203.0.113.1", ib)
		encoded := strings.Split(strings.SplitN(strings.TrimPrefix(uri, "ss://"), "@", 2)[0], "#")[0]
		raw, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil || string(raw) != tc.method+":"+want {
			t.Errorf("bad URI for %s", tc.method)
		}
		clash := shadowsocksClash(user, "203.0.113.1", ib)
		if !strings.Contains(clash, "password: "+"\""+want+"\"") || !strings.Contains(clash, "cipher: "+tc.method) {
			t.Errorf("bad Mihomo output for %s", tc.method)
		}
		var outbound map[string]any
		if err := json.Unmarshal([]byte(shadowsocksSingBox(user, "203.0.113.1", ib)), &outbound); err != nil {
			t.Fatal(err)
		}
		if outbound["password"] != want || outbound["method"] != tc.method {
			t.Errorf("sing-box password mismatch for %s", tc.method)
		}
		if (legacyUnsupportedReason(ib) == "") != tc.legacy {
			t.Errorf("wrong legacy compatibility for %s", tc.method)
		}
		if (shadowsocksSurge(user, "203.0.113.1", ib) != "") != tc.surge {
			t.Errorf("wrong Surge compatibility for %s", tc.method)
		}
		if user.Password != "different-trojan-password" {
			t.Fatal("SS modified other protocol password")
		}
	}
}

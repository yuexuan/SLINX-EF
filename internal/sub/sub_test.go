package sub

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/slinxlink/node/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestManagedEgressCredentialsNeverEnterUserSubscriptions(t *testing.T) {
	previous := database.DB
	defer func() { database.DB = previous }()
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB.AutoMigrate(&database.Config{}, &database.User{}, &database.Inbound{}, &database.Endpoint{})
	database.DB.Create(&database.Config{IPv4: "203.0.113.1"})
	inbound := database.Inbound{Enable: true, Name: "Reality-entry", Protocol: "vless", Port: 443, TLSType: "reality", RealityServerName: "www.cloudflare.com", RealityShortIDs: `["abcd"]`, RealityPublicKey: "public-key", Flow: "xtls-rprx-vision"}
	database.DB.Create(&inbound)
	ids, _ := json.Marshal([]uint{inbound.ID})
	user := database.User{Enable: true, Name: "subscriber", Token: "test-token", Inbounds: string(ids), UUID: "11111111-1111-4111-8111-111111111111"}
	database.DB.Create(&user)
	database.DB.Create(&database.Endpoint{Enable: true, Tag: "ISP", Type: "ssh", Username: "ISP-PRIVATE-USERNAME", Password: "ISP-PRIVATE-PASSWORD", Options: `{"private_key":"ISP-PRIVATE-KEY"}`, Dynamic: `{"headers":{"Authorization":"PROVIDER-PRIVATE-TOKEN"}}`})
	outputs := []string{Sub(user.Token), Uri(user, inbound), Json(user, inbound, "singbox")}
	clash, _ := Clash(user.Token)
	surge, _ := Surge(user.Token)
	outputs = append(outputs, clash, surge)
	for _, out := range outputs {
		for _, secret := range []string{"ISP-PRIVATE-USERNAME", "ISP-PRIVATE-PASSWORD", "ISP-PRIVATE-KEY", "PROVIDER-PRIVATE-TOKEN"} {
			if strings.Contains(out, secret) {
				t.Fatal("managed credentials leaked into subscriber output")
			}
		}
	}
	if !strings.Contains(outputs[0], user.UUID) || !strings.Contains(outputs[0], "203.0.113.1:443") || !strings.Contains(outputs[0], "security=reality") {
		t.Fatal("Reality entry subscription regressed")
	}
}

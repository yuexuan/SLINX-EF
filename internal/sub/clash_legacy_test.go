package sub

import (
	"testing"

	"github.com/slinxlink/node/internal/database"
)

func TestLegacyClashInboundCompatibility(t *testing.T) {
	cases := []struct {
		inbound database.Inbound
		allowed bool
	}{
		{database.Inbound{Protocol: "vmess", TLSType: "none", Transport: "raw"}, true},
		{database.Inbound{Protocol: "vmess", TLSType: "tls", Transport: "websocket"}, true},
		{database.Inbound{Protocol: "trojan", TLSType: "tls", Transport: "raw"}, true},
		{database.Inbound{Protocol: "vless", TLSType: "reality", Transport: "raw"}, false},
		{database.Inbound{Protocol: "hysteria", TLSType: "tls"}, false},
		{database.Inbound{Protocol: "anytls", TLSType: "tls"}, false},
		{database.Inbound{Protocol: "vmess", TLSType: "reality"}, false},
		{database.Inbound{Protocol: "vmess", TLSType: "none", ECHEnabled: true}, false},
		{database.Inbound{Protocol: "trojan", TLSType: "none"}, false},
	}
	for _, tc := range cases {
		allowed := legacyUnsupportedReason(tc.inbound) == ""
		if allowed != tc.allowed {
			t.Errorf("protocol=%s TLS=%s: got allowed=%t, want %t", tc.inbound.Protocol, tc.inbound.TLSType, allowed, tc.allowed)
		}
	}
}

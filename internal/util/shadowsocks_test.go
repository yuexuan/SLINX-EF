package util

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestShadowsocksKeys(t *testing.T) {
	methods := []struct {
		method string
		bytes  int
	}{
		{"aes-128-gcm", 0},
		{"aes-256-gcm", 0},
		{"chacha20-ietf-poly1305", 0},
		{"2022-blake3-aes-128-gcm", 16},
		{"2022-blake3-aes-256-gcm", 32},
	}
	for _, tc := range methods {
		key, err := GenerateShadowsocksPassword(tc.method)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateShadowsocksPassword(tc.method, key); err != nil {
			t.Errorf("%s: %v", tc.method, err)
		}
		client := ShadowsocksClientPassword(tc.method, key, "token-1")
		another := ShadowsocksClientPassword(tc.method, key, "token-2")
		if client == another {
			t.Errorf("different users share key with %s", tc.method)
		}
		if client != ShadowsocksClientPassword(tc.method, key, "token-1") {
			t.Errorf("unstable key for %s", tc.method)
		}
		if tc.bytes > 0 {
			if !strings.HasPrefix(client, key+":") {
				t.Errorf("missing server key for %s", tc.method)
			}
			parts := strings.Split(client, ":")
			if len(parts) != 2 {
				t.Errorf("bad 2022 password %s", tc.method)
				continue
			}
			for _, part := range parts {
				bytes, err := base64.StdEncoding.Strict().DecodeString(part)
				if err != nil || len(bytes) != tc.bytes {
					t.Errorf("bad 2022 key %s: %v", tc.method, err)
				}
			}
		} else if strings.Contains(client, ":") {
			t.Errorf("legacy password contains colon for %s", tc.method)
		}
	}
	if ValidateShadowsocksPassword("2022-blake3-chacha20-poly1305", "key") == nil {
		t.Fatal("pinned sing-box rejected 2022 ChaCha20 multi-user mode")
	}
	if ValidateShadowsocksPassword("unsupported", "key") == nil {
		t.Fatal("unsupported cipher accepted")
	}
	if ValidateShadowsocksPassword("2022-blake3-aes-128-gcm", "password") == nil {
		t.Fatal("2022 accepted invalid key")
	}
}

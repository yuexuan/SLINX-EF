package template

import (
	"strings"
	"testing"
)

func TestClashTemplateLegacyRules(t *testing.T) {
	rs := &Ruleset{Groups: []RuleGroup{
		{Name: "SOCIAL", Rules: []string{"DOMAIN-SUFFIX,telegram.org", "DOMAIN-SUFFIX,91.108.4.0/22", "DOMAIN-SUFFIX,2001:67c:4e8::/48"}},
		{Name: "_final"},
	}}
	modern := renderClashTemplate(rs, false)
	legacy := renderClashTemplate(rs, true)
	if !strings.Contains(modern, "GEOSITE,CN") {
		t.Fatal("modern Clash lost GEOSITE support")
	}
	if strings.Contains(legacy, "GEOSITE,") || strings.Contains(legacy, "GEOIP,LAN") {
		t.Fatal("legacy Clash must not contain unsupported extended rules")
	}
	for _, value := range []string{
		"IP-CIDR,91.108.4.0/22,💬 通讯,no-resolve",
		"IP-CIDR6,2001:67c:4e8::/48,💬 通讯,no-resolve",
		"IP-CIDR,192.168.0.0/16,🎯 直连,no-resolve",
		"GEOIP,CN,🎯 直连,no-resolve",
		"MATCH,🐟 默认",
	} {
		if !strings.Contains(legacy, value) {
			t.Fatalf("legacy template missing rule %q", value)
		}
	}
}

func TestClashIPPrefixRecognition(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"DOMAIN-SUFFIX,91.108.4.0/22", "IP-CIDR,91.108.4.0/22"},
		{"DOMAIN-SUFFIX,2001:67c:4e8::/48", "IP-CIDR6,2001:67c:4e8::/48"},
		{"DOMAIN-SUFFIX,example.org", "DOMAIN-SUFFIX,example.org"},
	} {
		if got := normalizeClashRule(tc.input); got != tc.want {
			t.Errorf("normalizeClashRule(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}

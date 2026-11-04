package config

import (
	"slices"
	"testing"
)

// An unset or empty TRUSTED_PROXIES must trust the private ranges, so a
// deployment behind a Docker bridge gateway or a local proxy works without any
// configuration.
func TestParseTrustedProxiesDefaults(t *testing.T) {
	for _, raw := range []string{"", "   ", "\n"} {
		got := parseTrustedProxies(raw)
		if !slices.Equal(got, defaultTrustedProxies) {
			t.Fatalf("parseTrustedProxies(%q) = %v, want the private-range defaults", raw, got)
		}
	}
}

// "none" is the explicit way back to trusting nobody.
func TestParseTrustedProxiesNone(t *testing.T) {
	for _, raw := range []string{"none", "NONE", " none "} {
		if got := parseTrustedProxies(raw); len(got) != 0 {
			t.Fatalf("parseTrustedProxies(%q) = %v, want no trusted proxy", raw, got)
		}
	}
}

func TestParseTrustedProxiesOverride(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"cidr and ip", "172.28.0.0/24, 198.51.100.7", []string{"172.28.0.0/24", "198.51.100.7"}},
		{"ipv6 cidr", "2001:db8::/32", []string{"2001:db8::/32"}},
		{"blank entries skipped", "172.28.0.0/24,,", []string{"172.28.0.0/24"}},
		{"invalid entries dropped", "nonsense,172.28.0.0/24", []string{"172.28.0.0/24"}},
		// Everything invalid leaves nothing trusted rather than silently
		// falling back to the permissive default.
		{"all invalid", "nonsense", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTrustedProxies(tc.raw)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("parseTrustedProxies(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

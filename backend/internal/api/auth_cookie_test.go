package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/store"
)

func newRouterWithProxies(t *testing.T, proxies []string) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	return NewRouter(config.Config{
		JWTSecret:         "test-secret",
		TokenTTL:          time.Hour,
		AllowRegistration: true,
		Version:           "test",
		TrustedProxies:    proxies,
	}, st)
}

// registerFrom registers a user from a request whose RemoteAddr and
// X-Forwarded-Proto header are set, and returns the issued session cookie.
func registerFrom(t *testing.T, router http.Handler, remoteAddr, forwardedProto string) *http.Cookie {
	t.Helper()
	body := `{"email":"cookie@example.com","password":"supersecret","name":"Cookie"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	if forwardedProto != "" {
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("register: expected a session cookie")
	}
	return cookies[0]
}

// X-Forwarded-Proto is client-controlled on a direct connection: only a request
// that arrives over TLS, or one that came through a TRUSTED_PROXIES hop, may
// produce a Secure cookie.
func TestSessionCookieSecuredOnlyByTLSOrTrustedProxy(t *testing.T) {
	const remoteAddr = "192.0.2.20:54321"

	t.Run("untrusted header is ignored", func(t *testing.T) {
		router := newRouterWithProxies(t, nil)
		cookie := registerFrom(t, router, remoteAddr, "https")
		if cookie.Secure {
			t.Fatal("X-Forwarded-Proto from an untrusted client must not mark the cookie Secure")
		}
	})

	t.Run("trusted proxy outside the CIDR is ignored", func(t *testing.T) {
		router := newRouterWithProxies(t, []string{"198.51.100.0/24"})
		cookie := registerFrom(t, router, remoteAddr, "https")
		if cookie.Secure {
			t.Fatal("a header from a peer outside TRUSTED_PROXIES must not mark the cookie Secure")
		}
	})

	t.Run("trusted CIDR with https header", func(t *testing.T) {
		router := newRouterWithProxies(t, []string{"192.0.2.0/24"})
		cookie := registerFrom(t, router, remoteAddr, "https")
		if !cookie.Secure {
			t.Fatal("X-Forwarded-Proto: https from a trusted proxy should mark the cookie Secure")
		}
	})

	t.Run("trusted single IP with https header", func(t *testing.T) {
		router := newRouterWithProxies(t, []string{"192.0.2.20"})
		cookie := registerFrom(t, router, remoteAddr, "https")
		if !cookie.Secure {
			t.Fatal("a trusted proxy IP should mark the cookie Secure")
		}
	})

	t.Run("trusted proxy without the header", func(t *testing.T) {
		router := newRouterWithProxies(t, []string{"192.0.2.0/24"})
		cookie := registerFrom(t, router, remoteAddr, "")
		if cookie.Secure {
			t.Fatal("a trusted proxy on plain HTTP must not mark the cookie Secure")
		}
	})

	t.Run("direct TLS", func(t *testing.T) {
		router := newRouterWithProxies(t, nil)

		body := `{"email":"tls@example.com","password":"supersecret","name":"TLS"}`
		req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = remoteAddr
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register over TLS: expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		cookies := rec.Result().Cookies()
		if len(cookies) == 0 || !cookies[0].Secure {
			t.Fatal("a direct TLS request should always get a Secure cookie")
		}
	})
}

// isTrustedProxy must accept plain IPs and CIDR blocks in both address families.
func TestIsTrustedProxy(t *testing.T) {
	cases := []struct {
		name    string
		remote  string
		trusted []string
		want    bool
	}{
		{"ipv4 cidr match", "192.0.2.7:1234", []string{"192.0.2.0/24"}, true},
		{"ipv4 cidr miss", "203.0.113.7:1234", []string{"192.0.2.0/24"}, false},
		{"ipv4 exact match", "192.0.2.7:1234", []string{"192.0.2.7"}, true},
		{"ipv4 exact miss", "192.0.2.8:1234", []string{"192.0.2.7"}, false},
		{"ipv6 cidr match", "[2001:db8::1]:1234", []string{"2001:db8::/32"}, true},
		{"ipv6 cidr miss", "[2001:dba::1]:1234", []string{"2001:db8::/32"}, false},
		{"ipv6 exact match", "[2001:db8::1]:1234", []string{"2001:db8::1"}, true},
		{"ipv6 loopback", "[::1]:1234", []string{"::1"}, true},
		{"no port", "192.0.2.7", []string{"192.0.2.0/24"}, true},
		{"empty trusted list", "192.0.2.7:1234", nil, false},
		{"unparsable remote", "not-an-ip", []string{"192.0.2.0/24"}, false},
		{"malformed entry skipped", "192.0.2.7:1234", []string{"nonsense", "192.0.2.0/24"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTrustedProxy(tc.remote, tc.trusted); got != tc.want {
				t.Fatalf("isTrustedProxy(%q, %v) = %v, want %v", tc.remote, tc.trusted, got, tc.want)
			}
		})
	}
}

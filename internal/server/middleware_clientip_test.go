package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// runClientIP drives a request through clientIPMiddleware (built from the given
// trusted proxy CIDRs) and returns what clientIP resolves inside the handler.
func runClientIP(t *testing.T, cidrs []string, remoteAddr, xff string) string {
	t.Helper()
	srv := &Server{trustedProxyCIDRs: cidrs}
	var got string
	h := srv.clientIPMiddleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientIP(r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

// With no trusted proxies the client IP is the TCP connection address and any
// client-supplied X-Forwarded-For is ignored — the spoofing fix versus RealIP.
func TestClientIP_RemoteAddrMode_IgnoresXFF(t *testing.T) {
	got := runClientIP(t, nil, "203.0.113.7:5555", "1.2.3.4")
	assert.Equal(t, "203.0.113.7", got)
}

// Behind a declared proxy, the client IP is taken from X-Forwarded-For.
func TestClientIP_XFFMode_ExtractsClientBehindProxy(t *testing.T) {
	got := runClientIP(t, []string{"10.0.0.0/8"}, "10.0.0.1:443", "198.51.100.23")
	assert.Equal(t, "198.51.100.23", got)
}

// A client that prepends a forged X-Forwarded-For entry cannot win: the proxy
// appends the real connection IP to the right, and the walk skips only trusted
// hops, so the forged leftmost value is never reached.
func TestClientIP_XFFMode_SpoofedEntryIgnored(t *testing.T) {
	got := runClientIP(t, []string{"10.0.0.0/8"}, "10.0.0.1:443", "1.1.1.1, 198.51.100.23")
	assert.Equal(t, "198.51.100.23", got)
}

// clientIP falls back to the raw connection address when no middleware ran.
func TestClientIP_FallbackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.55:1111"
	assert.Equal(t, "192.0.2.55", clientIP(req))
}

// When RemoteAddr has no host:port form (no middleware ran), clientIP returns it
// verbatim rather than dropping it.
func TestClientIP_FallbackRemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "bare-no-port"
	assert.Equal(t, "bare-no-port", clientIP(req))
}

// WithTrustedProxyCIDRs is applied by NewServer and selects XFF extraction.
func TestWithTrustedProxyCIDRs(t *testing.T) {
	plain := NewServer(nil, nil, "test")
	assert.Nil(t, plain.trustedProxyCIDRs)

	srv := NewServer(nil, nil, "test", WithTrustedProxyCIDRs([]string{"10.0.0.0/8"}))
	assert.Equal(t, []string{"10.0.0.0/8"}, srv.trustedProxyCIDRs)

	// The option must actually drive extraction: a client behind the declared
	// proxy is read from X-Forwarded-For.
	var got string
	h := srv.clientIPMiddleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientIP(r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.23")
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "198.51.100.23", got)
}

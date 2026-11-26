package handler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/internal/utils"
)

// fakeDNS makes the given names resolve to fixed addresses, so policy tests do
// not depend on the machine's DNS. Names not listed use the real resolver.
func fakeDNS(t *testing.T, records map[string][]string) {
	t.Helper()
	orig := utils.LookupIP
	utils.LookupIP = func(host string) ([]net.IP, error) {
		if addrs, ok := records[host]; ok {
			var out []net.IP
			for _, a := range addrs {
				out = append(out, net.ParseIP(a))
			}
			return out, nil
		}
		return orig(host)
	}
	t.Cleanup(func() { utils.LookupIP = orig })
}

// Wildcards work through the proxy, not only in the matcher: *.sub.test covers
// its subdomains at any depth, but not the apex.
func TestWildcardThroughProxy(t *testing.T) {
	fakeDNS(t, map[string][]string{
		"api.sub.test":      {"127.0.0.1"},
		"deep.api.sub.test": {"127.0.0.1"},
		"sub.test":          {"127.0.0.1"},
	})
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"*.sub.test"}})
	proxy := srv.Listener.Addr().String()

	expectStatus(t, proxy, "api.sub.test:"+port, 200, "subdomain covered by wildcard")
	expectStatus(t, proxy, "deep.api.sub.test:"+port, 200, "deep subdomain covered by wildcard")
	expectStatus(t, proxy, "sub.test:"+port, 403, "apex not covered by *.sub.test")
}

// Internationalized names: the policy may be written in Unicode, and clients
// send the punycode form (xn--) in CONNECT, which must match it.
func TestInternationalizedNameThroughProxy(t *testing.T) {
	fakeDNS(t, map[string][]string{
		"xn--mnchen-3ya.test": {"127.0.0.1"},
	})
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"münchen.test"}})
	expectStatus(t, srv.Listener.Addr().String(), "xn--mnchen-3ya.test:"+port, 200, "punycode request matches unicode policy entry")
	expectStatus(t, srv.Listener.Addr().String(), "xn--mnchen-3ya.de:"+port, 403, "different domain not matched")
}

// Malformed CONNECT targets are rejected as bad requests before any policy check.
func TestMalformedConnectRejected(t *testing.T) {
	srv := startProxy(t, config.ACL{Default: "permit"})
	for _, target := range []string{"example.com", "", "example.com:"} {
		got, c, _ := connect(t, srv.Listener.Addr().String(), target)
		c.Close()
		if got != 400 {
			t.Errorf("CONNECT %q: got %d, want 400", target, got)
		}
	}
}

// A destination that does not answer within dial_timeout gets 503, and the
// client is not left waiting.
func TestDestinationTimeoutReturns503(t *testing.T) {
	// 192.0.2.1 is TEST-NET-1 (documentation): it does not answer SYNs.
	config.Set(config.Config{
		Logger: false, DialTimeout: 1,
		ACL:  config.ACL{Default: "permit"},
		Auth: config.Auth{Users: map[string]string{}},
	})
	srv := httptest.NewServer(ProxyHandler(http.NotFoundHandler()))
	t.Cleanup(srv.Close)

	start := time.Now()
	got, c, _ := connect(t, srv.Listener.Addr().String(), "192.0.2.1:443")
	c.Close()
	elapsed := time.Since(start)
	if got != 503 {
		t.Errorf("unresponsive destination: got %d, want 503", got)
	}
	if elapsed > 5*time.Second {
		t.Errorf("client waited %v; dial_timeout is 1s", elapsed)
	}
	t.Logf("unresponsive destination answered in %v", elapsed)
}

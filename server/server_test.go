package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/juliotorresmoreno/doppler/config"
)

// A request whose headers exceed max_header_bytes is refused by the server; a
// normal request is not.
func TestMaxHeaderBytesIsEnforced(t *testing.T) {
	config.Set(config.Config{Addr: "127.0.0.1:0", MaxHeaderBytes: 512, ACL: config.ACL{Default: "permit"}, ReadTimeout: 5, WriteTimeout: 5})
	svr, _ := Configure()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go svr.Serve(ln)
	t.Cleanup(func() { svr.Close() })
	base := "http://" + ln.Addr().String()

	small, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	small.Body.Close()
	if small.StatusCode == http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("normal request refused")
	}

	req, _ := http.NewRequest("GET", base+"/", nil)
	req.Header.Set("X-Big", strings.Repeat("a", 64<<10))
	big, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	big.Body.Close()
	if big.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Errorf("oversized headers: status %d, want 431", big.StatusCode)
	}
}

func pacSetup(t *testing.T, template string, extra func(*config.Config)) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "proxy.pac"), []byte(template), 0o600); err != nil {
		t.Fatal(err)
	}
	config.SetPath(filepath.Join(dir, "config.yml"))
	c := config.Config{TLSAddr: ":5443", ACL: config.ACL{Default: "permit"}}
	if extra != nil {
		extra(&c)
	}
	config.Set(c)
}

const pacTemplate = `function FindProxyForURL(url, host) { return "HTTPS {{.Host}}:{{.TLSPort}}"; }`

func TestPACServedOnTLSAndLoopbackOnly(t *testing.T) {
	pacSetup(t, pacTemplate, nil)

	get := func(remote string, tlsOn bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "http://proxy.example.com:5000/proxy.pac", nil)
		req.RemoteAddr = remote
		if tlsOn {
			req.TLS = &tls.ConnectionState{}
		}
		rec := httptest.NewRecorder()
		servePAC(rec, req)
		return rec
	}

	tlsRec := get("192.168.1.9:40000", true)
	if tlsRec.Code != 200 || !strings.Contains(tlsRec.Body.String(), "HTTPS proxy.example.com:5443") {
		t.Errorf("TLS PAC: %d %q", tlsRec.Code, tlsRec.Body.String())
	}
	if ct := tlsRec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("PAC content type %q; browsers must display it, not download it", ct)
	}

	if rec := get("127.0.0.1:40000", false); rec.Code != 200 {
		t.Errorf("loopback plain PAC: %d, want 200", rec.Code)
	}
	if rec := get("192.168.1.9:40000", false); rec.Code != 404 {
		t.Errorf("remote plain PAC: %d, want 404", rec.Code)
	}
}

func TestPACUsesPublicHostWhenSet(t *testing.T) {
	pacSetup(t, pacTemplate, func(c *config.Config) { c.PublicHost = "vpn.example.org" })
	req := httptest.NewRequest("GET", "/proxy.pac", nil)
	req.RemoteAddr = "10.0.0.1:1"
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	servePAC(rec, req)
	if !strings.Contains(rec.Body.String(), "HTTPS vpn.example.org:5443") {
		t.Errorf("public_host not used: %q", rec.Body.String())
	}
}

func TestPACMissingFileIs404(t *testing.T) {
	config.SetPath(filepath.Join(t.TempDir(), "config.yml"))
	config.Set(config.Config{TLSAddr: ":5443", ACL: config.ACL{Default: "permit"}})
	req := httptest.NewRequest("GET", "/proxy.pac", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	servePAC(rec, req)
	if rec.Code != 404 {
		t.Errorf("missing PAC: %d, want 404", rec.Code)
	}
}

// The metrics listener serves /metrics and nothing else; it is off when
// metrics_addr is empty.
func TestMetricsListenerServesOnlyMetrics(t *testing.T) {
	config.Set(config.Config{MetricsAddr: "127.0.0.1:9100", ACL: config.ACL{Default: "permit"}})
	svr := ConfigureMetrics()
	if svr == nil || svr.Addr != "127.0.0.1:9100" {
		t.Fatalf("metrics server not configured: %+v", svr)
	}

	ok := httptest.NewRecorder()
	svr.Handler.ServeHTTP(ok, httptest.NewRequest("GET", "/metrics", nil))
	if ok.Code != 200 || !strings.Contains(ok.Body.String(), "doppler_connections_active") {
		t.Errorf("/metrics: %d", ok.Code)
	}

	other := httptest.NewRecorder()
	svr.Handler.ServeHTTP(other, httptest.NewRequest("GET", "/proxy.pac", nil))
	if other.Code != 404 {
		t.Errorf("other path on metrics listener: %d, want 404", other.Code)
	}

	config.Set(config.Config{ACL: config.ACL{Default: "permit"}})
	if ConfigureMetrics() != nil {
		t.Error("metrics listener should be off when metrics_addr is empty")
	}
}

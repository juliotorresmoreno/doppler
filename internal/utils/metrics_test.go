package utils

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/juliotorresmoreno/doppler/config"
)

func resetMetrics() {
	metricsMu.Lock()
	counters = map[string]map[string]int64{}
	metricsMu.Unlock()
	ActiveConnections.Store(0)
}

func TestObserveAccessCountsDecisionsAndBytes(t *testing.T) {
	resetMetrics()
	ev := NewAccessEvent("connect", "1.2.3.4", "CONNECT", "example.com:443")
	ev.SetDecision(true, ReasonPermitName, nil)
	ev.BytesUp, ev.BytesDown = 10, 20
	ObserveAccess(ev)

	denied := NewAccessEvent("http", "1.2.3.4", "GET", "blocked.example:80")
	denied.SetDecision(false, ReasonBlockName, nil)
	ObserveAccess(denied)

	authFail := NewAccessEvent("connect", "1.2.3.4", "CONNECT", "x:443")
	authFail.SetDecision(false, "auth_failed", nil)
	ObserveAccess(authFail)

	dns := NewAccessEvent("connect", "1.2.3.4", "CONNECT", "nx.invalid:443")
	dns.SetDecision(false, ReasonDNSFailure, nil)
	ObserveAccess(dns)

	var b strings.Builder
	WriteMetrics(&b)
	out := b.String()
	for _, want := range []string{
		`doppler_requests_total{event="connect"} 3`,
		`doppler_requests_total{event="http"} 1`,
		`doppler_decisions_total{decision="allow",reason="permit_name"} 1`,
		`doppler_decisions_total{decision="deny",reason="block_name"} 1`,
		`doppler_decisions_total{decision="deny",reason="dns_failure"} 1`,
		`doppler_auth_failures_total{reason="auth_failed"} 1`,
		`doppler_bytes_total{direction="up"} 10`,
		`doppler_bytes_total{direction="down"} 20`,
		`doppler_connections_active 0`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %q\n%s", want, out)
		}
	}
}

func TestMetricsAllowList(t *testing.T) {
	resetMetrics()
	scrape := func(allow []string, remote string) int {
		config.Set(config.Config{MetricsAllow: allow})
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.RemoteAddr = remote
		rec := httptest.NewRecorder()
		MetricsHandler().ServeHTTP(rec, req)
		if rec.Code == 200 && !strings.Contains(rec.Body.String(), "doppler_connections_active") {
			t.Errorf("200 without metrics body")
		}
		return rec.Code
	}
	if c := scrape(nil, "10.0.0.5:50000"); c != 200 {
		t.Errorf("empty allow list should serve any client, got %d", c)
	}
	allow := []string{"10.0.0.0/8"}
	if c := scrape(allow, "10.9.8.7:50000"); c != 200 {
		t.Errorf("client in allowed range: %d", c)
	}
	if c := scrape(allow, "192.168.1.5:50000"); c != 404 {
		t.Errorf("client outside allowed range: %d, want 404", c)
	}
}

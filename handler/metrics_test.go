package handler

import (
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/internal/utils"
)

// metricValue returns the value of an exact metric line, or -1 if absent.
func metricValue(t *testing.T, line string) int64 {
	t.Helper()
	var b strings.Builder
	utils.WriteMetrics(&b)
	for _, l := range strings.Split(b.String(), "\n") {
		if strings.HasPrefix(l, line+" ") {
			var v int64
			for _, c := range strings.TrimPrefix(l, line+" ") {
				v = v*10 + int64(c-'0')
			}
			return v
		}
	}
	return -1
}

func TestMetricsCountAllowedAndDeniedTunnels(t *testing.T) {
	addr, _ := target(t, func(c net.Conn) { io.Copy(io.Discard, c) })
	port := portOf(t, addr)
	allowedBefore := metricValue(t, `doppler_decisions_total{decision="allow",reason="permit_name"}`)
	deniedBefore := metricValue(t, `doppler_decisions_total{decision="deny",reason="default_block"}`)

	config.Set(config.Config{Logger: false, ACL: config.ACL{Default: "block", Permit: []string{"127.0.0.1"}}, Auth: config.Auth{Users: map[string]string{}}})
	srv := httptest.NewServer(ProxyHandler(nil))
	t.Cleanup(srv.Close)

	got, c, _ := connect(t, srv.Listener.Addr().String(), "127.0.0.1:"+port)
	c.Close()
	if got != 200 {
		t.Fatalf("allowed tunnel: %d", got)
	}
	got, c, _ = connect(t, srv.Listener.Addr().String(), "example.org:443")
	c.Close()
	if got != 403 {
		t.Fatalf("denied tunnel: %d", got)
	}

	if v := metricValue(t, `doppler_decisions_total{decision="allow",reason="permit_name"}`); v != allowedBefore+1 {
		t.Errorf("allowed decisions: %d, want %d", v, allowedBefore+1)
	}
	if v := metricValue(t, `doppler_decisions_total{decision="deny",reason="default_block"}`); v != deniedBefore+1 {
		t.Errorf("denied decisions: %d, want %d", v, deniedBefore+1)
	}
}

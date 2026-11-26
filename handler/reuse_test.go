package handler

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
)

// Forwarded requests do not share connections, and none stay open after the
// request ends: the proxy must not leave idle destination sockets behind.
func TestForwardedRequestsCloseTheirConnections(t *testing.T) {
	var opened, closed int64
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	backend.Config.ConnState = func(c net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			atomic.AddInt64(&opened, 1)
		case http.StateClosed, http.StateHijacked:
			atomic.AddInt64(&closed, 1)
		}
	}
	backend.Start()
	defer backend.Close()
	port := portOf(t, backend.Listener.Addr().String())
	srv := startProxy(t, config.ACL{Default: "permit"})
	pu, _ := url.Parse("http://" + srv.Listener.Addr().String())
	c := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(pu)}}

	for i := 0; i < 20; i++ {
		resp, err := c.Get("http://127.0.0.1:" + port + "/")
		if err != nil {
			t.Fatal(err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadInt64(&closed) < atomic.LoadInt64(&opened) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if open := atomic.LoadInt64(&opened) - atomic.LoadInt64(&closed); open > 0 {
		t.Errorf("%d destination connections left open after 20 forwarded requests", open)
	}
}

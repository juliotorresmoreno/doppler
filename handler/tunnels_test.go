package handler

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
)

func TestTunnelRegistryRefusesAfterClose(t *testing.T) {
	defer tunnels.reopen()
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	release, ok := tunnels.track(a, b)
	if !ok {
		t.Fatal("tunnel should be tracked before shutdown")
	}
	release()

	CloseTunnels()
	c, d := net.Pipe()
	defer c.Close()
	defer d.Close()
	if _, ok := tunnels.track(c, d); ok {
		t.Error("tunnel accepted after CloseTunnels")
	}
}

// CloseTunnels must end an open tunnel: the client sees the connection close
// instead of hanging.
func TestCloseTunnelsEndsOpenTunnel(t *testing.T) {
	defer tunnels.reopen()
	addr, _ := target(t, func(c net.Conn) { io.Copy(io.Discard, c) })
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, 0, 0)

	got, c, br := connect(t, srv.Listener.Addr().String(), addr)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	defer c.Close()

	// Give the tunnel a moment to register, then shut down.
	time.Sleep(100 * time.Millisecond)
	CloseTunnels()

	c.SetDeadline(time.Now().Add(3 * time.Second))
	_, err := io.ReadAll(br)
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("open tunnel was not closed by CloseTunnels")
	}
}

package handler

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
)

// startProxyWithTimeouts is like startProxy but sets the HTTP server's read and
// write timeouts, as the real server does from the config.
func startProxyWithTimeouts(t *testing.T, acl config.ACL, read, write time.Duration) *httptest.Server {
	t.Helper()
	config.Set(config.Config{Logger: false, ACL: acl, Auth: config.Auth{Users: map[string]string{}}})
	srv := httptest.NewUnstartedServer(ProxyHandler(http.NotFoundHandler()))
	srv.Config.ReadTimeout = read
	srv.Config.WriteTimeout = write
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

// target accepts connections and reports, per connection, when the server side
// sees the peer close (read returns an error).
func target(t *testing.T, handle func(c net.Conn)) (addr string, closed chan struct{}) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	closed = make(chan struct{}, 100)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				handle(c)
				closed <- struct{}{}
			}(c)
		}
	}()
	return ln.Addr().String(), closed
}

// When the client goes away, the destination connection must be closed.
func TestLifecycleClientDisconnectClosesDestination(t *testing.T) {
	addr, closed := target(t, func(c net.Conn) {
		io.Copy(io.Discard, c) // returns when the proxy closes its side
	})
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, 0, 0)

	got, c, _ := connect(t, srv.Listener.Addr().String(), addr)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	c.Close()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("destination stayed open after the client disconnected")
	}
}

// When the destination goes away, the client must see the tunnel close.
func TestLifecycleDestinationCloseReachesClient(t *testing.T) {
	addr, _ := target(t, func(c net.Conn) {
		c.Write([]byte("bye"))
	})
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, 0, 0)

	got, c, br := connect(t, srv.Listener.Addr().String(), addr)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	data, err := io.ReadAll(br)
	if string(data) != "bye" {
		t.Errorf("client got %q (%v), want \"bye\" then EOF", data, err)
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Error("client was not closed after the destination closed")
	}
}

// Half-close: the client finishes sending, the destination must see EOF and
// can still reply through the tunnel.
func TestLifecycleHalfCloseKeepsReturnPath(t *testing.T) {
	addr, _ := target(t, func(c net.Conn) {
		req, _ := io.ReadAll(c) // reads until the proxy half-closes
		c.Write(append([]byte("reply:"), req...))
	})
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, 0, 0)

	got, c, br := connect(t, srv.Listener.Addr().String(), addr)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	defer c.Close()
	io.WriteString(c, "req")
	tcp, ok := c.(*net.TCPConn)
	if !ok {
		t.Skip("not a TCP connection")
	}
	tcp.CloseWrite()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	data, err := io.ReadAll(br)
	if string(data) != "reply:req" {
		t.Errorf("reply through half-closed tunnel: %q (%v)", data, err)
	}
}

// An idle tunnel must survive the HTTP server's read/write timeouts: they must
// not apply once the connection is hijacked.
func TestLifecycleTunnelSurvivesServerTimeouts(t *testing.T) {
	addr, _ := target(t, func(c net.Conn) { io.Copy(c, c) })
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, time.Second, time.Second)

	got, c, br := connect(t, srv.Listener.Addr().String(), addr)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	defer c.Close()
	time.Sleep(2500 * time.Millisecond)
	io.WriteString(c, "ping")
	c.SetDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br, buf); err != nil || string(buf) != "ping" {
		t.Errorf("tunnel cut by server timeout: %q %v", buf, err)
	}
}

// Closed tunnels must not leave goroutines behind.
func TestLifecycleNoGoroutineLeak(t *testing.T) {
	addr, _ := target(t, func(c net.Conn) { io.Copy(io.Discard, c) })
	srv := startProxyWithTimeouts(t, config.ACL{Default: "permit"}, 0, 0)

	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		got, c, _ := connect(t, srv.Listener.Addr().String(), addr)
		if got != 200 {
			t.Fatalf("tunnel %d: %d", i, got)
		}
		c.Close()
	}
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before+3 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > before+3 {
		t.Errorf("goroutines: before %d, after closed tunnels %d", before, n)
	}
}

var _ = bufio.NewReader

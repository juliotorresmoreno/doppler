package handler

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
)

// wsBackend answers a WebSocket upgrade with 101, then echoes bytes.
func wsBackend(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				req, err := http.ReadRequest(br)
				if err != nil || !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
					return
				}
				io.WriteString(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
				io.Copy(c, br) // echo everything after the handshake
			}(c)
		}
	}()
	return ln.Addr().String()
}

// A WebSocket upgrade through the proxy: the handshake reaches the backend, and
// data flows both ways afterwards.
func TestWebSocketUpgradeThroughProxy(t *testing.T) {
	backend := wsBackend(t)
	srv := startProxy(t, config.ACL{Default: "permit"})

	c, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	fmt.Fprintf(c, "GET http://%s/chat HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n", backend, backend)

	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, &http.Request{Method: "GET"})
	if err != nil {
		t.Fatalf("handshake response: %v", err)
	}
	if resp.StatusCode != 101 {
		t.Fatalf("handshake status %d, want 101", resp.StatusCode)
	}

	io.WriteString(c, "hello-ws")
	buf := make([]byte, len("hello-ws"))
	if _, err := io.ReadFull(br, buf); err != nil || string(buf) != "hello-ws" {
		t.Errorf("echo over upgraded connection: %q (%v)", buf, err)
	}
}

// The WebSocket path applies the same policy: a denied destination gets 403 and
// no handshake reaches it.
func TestWebSocketDeniedByPolicy(t *testing.T) {
	backend := wsBackend(t)
	srv := startProxy(t, config.ACL{Default: "block"})

	c, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(c, "GET http://%s/chat HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n", backend, backend)

	resp, err := http.ReadResponse(bufio.NewReader(c), &http.Request{Method: "GET"})
	if err != nil {
		t.Fatalf("response: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("denied WebSocket: status %d, want 403", resp.StatusCode)
	}
}

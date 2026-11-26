package handler

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// syncBuffer is a Writer safe for the handler goroutines and the test to share.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// captureAccess sends access events to a buffer for the duration of the test.
func captureAccess(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	orig := utils.AccessOutput
	utils.AccessOutput = buf
	t.Cleanup(func() { utils.AccessOutput = orig })
	return buf
}

// waitEvents waits until at least n access lines are written and returns them.
func waitEvents(t *testing.T, buf *syncBuffer, n int) []map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) >= n && lines[0] != "" {
			var out []map[string]any
			for _, l := range lines {
				var m map[string]any
				if err := json.Unmarshal([]byte(l), &m); err != nil {
					t.Fatalf("access line is not JSON: %q: %v", l, err)
				}
				out = append(out, m)
			}
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d access events, got: %q", n, buf.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func accessConfig(acl config.ACL, auth config.Auth) {
	config.Set(config.Config{Logger: false, AccessLog: true, ACL: acl, Auth: auth})
}

func TestAccessLogAllowedTunnelRecordsBytesAndReason(t *testing.T) {
	buf := captureAccess(t)
	addr, _ := target(t, func(c net.Conn) { io.Copy(c, c) }) // echo
	port := portOf(t, addr)
	accessConfig(config.ACL{Default: "block", Permit: []string{"127.0.0.1"}}, config.Auth{Users: map[string]string{}})
	srv := httptest.NewServer(ProxyHandler(http.NotFoundHandler()))
	t.Cleanup(srv.Close)

	got, c, br := connect(t, srv.Listener.Addr().String(), "127.0.0.1:"+port)
	if got != 200 {
		t.Fatalf("tunnel: %d", got)
	}
	io.WriteString(c, "hello")
	echo := make([]byte, 5)
	io.ReadFull(br, echo)
	c.Close()

	ev := waitEvents(t, buf, 1)[0]
	want := map[string]any{
		"event": "connect", "decision": "allow", "reason": "permit_name",
		"dest": "127.0.0.1:" + port, "status": float64(200),
		"bytes_up": float64(5), "bytes_down": float64(5),
	}
	for k, v := range want {
		if ev[k] != v {
			t.Errorf("%s = %v, want %v", k, ev[k], v)
		}
	}
	if res, _ := ev["resolved"].([]any); len(res) != 1 || res[0] != "127.0.0.1" {
		t.Errorf("resolved = %v", ev["resolved"])
	}
	if _, ok := ev["duration_ms"]; !ok {
		t.Error("duration_ms missing")
	}
	if ev["client"] == nil {
		t.Error("client missing")
	}
}

func TestAccessLogDeniedRecordsReasonAndStatus(t *testing.T) {
	buf := captureAccess(t)
	accessConfig(config.ACL{Default: "block"}, config.Auth{Users: map[string]string{}})
	srv := httptest.NewServer(ProxyHandler(http.NotFoundHandler()))
	t.Cleanup(srv.Close)

	got, c, _ := connect(t, srv.Listener.Addr().String(), "example.org:443")
	c.Close()
	if got != 403 {
		t.Fatalf("status %d", got)
	}
	ev := waitEvents(t, buf, 1)[0]
	if ev["decision"] != "deny" || ev["reason"] != "default_block" || ev["status"] != float64(403) {
		t.Errorf("denied event: %v", ev)
	}
	if ev["dest"] != "example.org:443" {
		t.Errorf("dest = %v", ev["dest"])
	}
}

// Credentials must never reach the log, but the user name is recorded.
func TestAccessLogRecordsUserWithoutSecrets(t *testing.T) {
	buf := captureAccess(t)
	addr, _ := target(t, func(c net.Conn) { io.Copy(io.Discard, c) })
	port := portOf(t, addr)
	hash, _ := bcrypt.GenerateFromPassword([]byte("S3cret-pw!"), bcrypt.MinCost)
	accessConfig(config.ACL{Default: "permit"}, config.Auth{Enabled: true, Users: map[string]string{"julio": string(hash)}})
	srv := httptest.NewUnstartedServer(ProxyHandler(http.NotFoundHandler()))
	srv.StartTLS()
	t.Cleanup(srv.Close)

	c, err := tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	creds := base64.StdEncoding.EncodeToString([]byte("julio:S3cret-pw!"))
	io.WriteString(c, "CONNECT 127.0.0.1:"+port+" HTTP/1.1\r\nHost: x\r\nProxy-Authorization: Basic "+creds+"\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(c), &http.Request{Method: "CONNECT"})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("authenticated CONNECT: %v %v", resp, err)
	}
	c.Close()

	ev := waitEvents(t, buf, 1)[0]
	if ev["user"] != "julio" || ev["decision"] != "allow" {
		t.Errorf("event = %v", ev)
	}
	raw := buf.String()
	for _, secret := range []string{"S3cret-pw!", creds, "Basic ", "Proxy-Authorization"} {
		if strings.Contains(raw, secret) {
			t.Errorf("access log leaks %q", secret)
		}
	}
}

func TestAccessLogFailedAuthRecordsReason(t *testing.T) {
	buf := captureAccess(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	accessConfig(config.ACL{Default: "permit"}, config.Auth{Enabled: true, Users: map[string]string{"julio": string(hash)}})
	srv := httptest.NewUnstartedServer(ProxyHandler(http.NotFoundHandler()))
	srv.StartTLS()
	t.Cleanup(srv.Close)

	c, _ := tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	bad := base64.StdEncoding.EncodeToString([]byte("julio:wrong"))
	io.WriteString(c, "CONNECT 127.0.0.1:9 HTTP/1.1\r\nHost: x\r\nProxy-Authorization: Basic "+bad+"\r\n\r\n")
	http.ReadResponse(bufio.NewReader(c), &http.Request{Method: "CONNECT"})
	c.Close()

	ev := waitEvents(t, buf, 1)[0]
	if ev["decision"] != "deny" || ev["reason"] != "auth_failed" || ev["user"] != nil {
		t.Errorf("failed-auth event: %v", ev)
	}
	if strings.Contains(buf.String(), "wrong") {
		t.Error("access log contains the wrong password")
	}
}

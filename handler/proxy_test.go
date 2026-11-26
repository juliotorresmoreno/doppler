package handler

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// startProxy runs the real proxy handler on a local server with the given ACL.
func startProxy(t *testing.T, acl config.ACL) *httptest.Server {
	t.Helper()
	config.Set(config.Config{
		Logger:       false,
		ACL:          acl,
		ReadTimeout:  30,
		WriteTimeout: 120,
		Auth:         config.Auth{Users: map[string]string{}},
	})
	srv := httptest.NewServer(ProxyHandler(http.NotFoundHandler()))
	t.Cleanup(srv.Close)
	return srv
}

// connect sends a CONNECT for target through the proxy at addr and returns the
// status code and the open connection (for tunnel tests).
func connect(t *testing.T, addr, target string, headers ...string) (int, net.Conn, *bufio.Reader) {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(20 * time.Second))
	req := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n"
	for _, h := range headers {
		req += h + "\r\n"
	}
	req += "\r\n"
	if _, err := io.WriteString(c, req); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
	if err != nil {
		c.Close()
		t.Fatalf("CONNECT %s: no response: %v", target, err)
	}
	return resp.StatusCode, c, br
}

func expectStatus(t *testing.T, addr, target string, want int, why string) {
	t.Helper()
	got, c, _ := connect(t, addr, target)
	c.Close()
	if got != want {
		t.Errorf("CONNECT %s: got %d, want %d (%s)", target, got, want, why)
	}
}

// echoServer accepts connections on ln and echoes each one back.
func echoServer(t *testing.T, network, addr string) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen(network, addr)
	if err != nil {
		t.Skipf("cannot listen on %s: %v", addr, err)
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
				io.Copy(c, c)
			}(c)
		}
	}()
	return ln, ln.Addr().String()
}

func portOf(t *testing.T, hostPort string) string {
	t.Helper()
	_, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatal(err)
	}
	return port
}

// --- Policy: domains, IPs and DNS ---

func TestExplicitlyPermittedDomain(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"localhost"}})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 200, "explicit permit")
}

func TestExplicitlyBlockedDomain(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit", Block: []string{"localhost"}})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 403, "explicit block")
}

func TestUnknownDomainDefaultDeny(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block"})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 403, "default deny")
}

func TestUnknownDomainDefaultPermit(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit"})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 200, "default permit")
}

func TestDomainInBothListsBlockWins(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit", Permit: []string{"localhost"}, Block: []string{"localhost"}})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 403, "block beats permit")
}

func TestCaseInsensitiveAndTrailingDot(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"LOCALHOST"}})
	expectStatus(t, srv.Listener.Addr().String(), "LocalHost.:"+port, 200, "case and trailing dot")
}

func TestDNSFailureDeniesUnlessPermitted(t *testing.T) {
	const nx = "doppler-nonexistent-host.invalid"
	srv := startProxy(t, config.ACL{Default: "permit"})
	expectStatus(t, srv.Listener.Addr().String(), nx+":443", 403, "unresolvable with default permit")
}

func TestDNSFailureWithExplicitPermitReportsDialError(t *testing.T) {
	const nx = "doppler-nonexistent-host.invalid"
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{nx}})
	expectStatus(t, srv.Listener.Addr().String(), nx+":443", 503, "permitted by name, dial fails")
}

// fakeResolver makes multi.test resolve to IPv4 and IPv6 loopback, so the
// multiple-address rules are tested without depending on the machine's DNS.
func fakeResolver(t *testing.T) {
	t.Helper()
	orig := utils.LookupIP
	utils.LookupIP = func(host string) ([]net.IP, error) {
		if host == "multi.test" {
			return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, nil
		}
		return orig(host)
	}
	t.Cleanup(func() { utils.LookupIP = orig })
}

func TestDomainWithMultipleAddressesBlockedByOneAddress(t *testing.T) {
	fakeResolver(t)
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	// The name is permitted, but one of its resolved addresses is blocked: block wins.
	srv := startProxy(t, config.ACL{Default: "permit", Permit: []string{"multi.test"}, Block: []string{"::1"}})
	expectStatus(t, srv.Listener.Addr().String(), "multi.test:"+port, 403, "one resolved address blocked")
}

func TestDomainWithMultipleAddressesPermittedByAnyAddress(t *testing.T) {
	fakeResolver(t)
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"127.0.0.1"}})
	expectStatus(t, srv.Listener.Addr().String(), "multi.test:"+port, 200, "a permitted resolved address allows the name")
}

func TestDirectIPConnect(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"127.0.0.1"}})
	expectStatus(t, srv.Listener.Addr().String(), "127.0.0.1:"+port, 200, "permitted IP literal")
	srv2 := startProxy(t, config.ACL{Default: "block"})
	expectStatus(t, srv2.Listener.Addr().String(), "127.0.0.1:"+port, 403, "IP literal under default deny")
}

func TestIPv4AndIPv6Destinations(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit", Block: []string{"127.0.0.0/8"}})
	expectStatus(t, srv.Listener.Addr().String(), "127.0.0.1:"+port, 403, "IPv4 CIDR block")

	if ln, err := net.Listen("tcp", "[::1]:0"); err == nil {
		ln.Close()
		_, v6 := echoServer(t, "tcp", "[::1]:0")
		v6port := portOf(t, v6)
		srv6 := startProxy(t, config.ACL{Default: "permit", Block: []string{"::1/128"}})
		expectStatus(t, srv6.Listener.Addr().String(), "[::1]:"+v6port, 403, "IPv6 CIDR block")
	} else {
		t.Log("IPv6 loopback unavailable; IPv6 destination test skipped")
	}
}

func TestIPv4MappedIPv6LiteralIsMatchedAsIPv4(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit", Block: []string{"127.0.0.0/8"}})
	expectStatus(t, srv.Listener.Addr().String(), "[::ffff:127.0.0.1]:"+port, 403, "IPv4-mapped IPv6 literal")
}

func TestCIDRBlockOnResolvedAddress(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit", Block: []string{"127.0.0.0/8"}})
	expectStatus(t, srv.Listener.Addr().String(), "localhost:"+port, 403, "resolved address inside blocked CIDR")
}

// --- Protocol handling ---

func TestHTTPForwardingUsesApprovedAddress(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok-from-backend")
	}))
	t.Cleanup(backend.Close)
	port := portOf(t, backend.Listener.Addr().String())
	srv := startProxy(t, config.ACL{Default: "block", Permit: []string{"localhost"}})

	proxyURL := "http://" + srv.Listener.Addr().String()
	c := &http.Client{Transport: &http.Transport{Proxy: func(*http.Request) (*url.URL, error) { return url.Parse(proxyURL) }}}
	resp, err := c.Get("http://localhost:" + port + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "ok-from-backend" {
		t.Errorf("forwarded GET: status %d body %q", resp.StatusCode, body)
	}
}

func TestInvalidConnectTargetIsRejected(t *testing.T) {
	srv := startProxy(t, config.ACL{Default: "permit"})
	for _, target := range []string{"[::bad", ":443"} {
		got, c, _ := connect(t, srv.Listener.Addr().String(), target)
		c.Close()
		if got == 200 {
			t.Errorf("invalid CONNECT target %q was accepted", target)
		}
	}
}

func TestDestinationRefusedReturns503(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().String()
	ln.Close()
	srv := startProxy(t, config.ACL{Default: "permit"})
	expectStatus(t, srv.Listener.Addr().String(), closed, 503, "nothing listening on destination")
}

func TestClientDisconnectClosesTunnel(t *testing.T) {
	ln, addr := echoServer(t, "tcp", "127.0.0.1:0")
	_ = ln
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit"})

	got, c, _ := connect(t, srv.Listener.Addr().String(), "127.0.0.1:"+port)
	if got != 200 {
		t.Fatalf("tunnel not established: %d", got)
	}
	c.Close()
	// A fresh tunnel must still work: the proxy did not stall on the closed one.
	got2, c2, br := connect(t, srv.Listener.Addr().String(), "127.0.0.1:"+port)
	defer c2.Close()
	if got2 != 200 {
		t.Fatalf("second tunnel after disconnect: %d", got2)
	}
	io.WriteString(c2, "ping")
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br, buf); err != nil || string(buf) != "ping" {
		t.Errorf("echo through tunnel failed: %q %v", buf, err)
	}
}

func TestConcurrentTunnels(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit"})

	const n = 25
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
			if err != nil {
				errs <- err
				return
			}
			defer c.Close()
			c.SetDeadline(time.Now().Add(15 * time.Second))
			fmt.Fprintf(c, "CONNECT 127.0.0.1:%s HTTP/1.1\r\nHost: x\r\n\r\n", port)
			br := bufio.NewReader(c)
			resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
			if err != nil || resp.StatusCode != 200 {
				errs <- fmt.Errorf("tunnel %d: %v", i, err)
				return
			}
			msg := "m" + strconv.Itoa(i)
			io.WriteString(c, msg)
			buf := make([]byte, len(msg))
			if _, err := io.ReadFull(br, buf); err != nil || string(buf) != msg {
				errs <- fmt.Errorf("tunnel %d echo: %q %v", i, buf, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// --- Authentication ---

func TestAuthRejectedOnPlainHTTP(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	config.Set(config.Config{ACL: config.ACL{Default: "permit"}, Auth: config.Auth{Enabled: true, Users: map[string]string{"julio": string(hash)}}})
	srv := httptest.NewServer(ProxyHandler(http.NotFoundHandler()))
	t.Cleanup(srv.Close)
	creds := "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("julio:pw"))
	got, c, _ := connect(t, srv.Listener.Addr().String(), "127.0.0.1:1", creds)
	c.Close()
	if got != 403 {
		t.Errorf("credentials over plain HTTP: got %d, want 403", got)
	}
}

func TestAuthOverTLS(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	config.Set(config.Config{ACL: config.ACL{Default: "permit"}, Auth: config.Auth{Enabled: true, Users: map[string]string{"julio": string(hash)}}})
	srv := httptest.NewUnstartedServer(ProxyHandler(http.NotFoundHandler()))
	srv.StartTLS()
	t.Cleanup(srv.Close)

	tlsConn := func(headers ...string) (int, net.Conn) {
		c, err := tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		c.SetDeadline(time.Now().Add(15 * time.Second))
		req := "CONNECT 127.0.0.1:" + port + " HTTP/1.1\r\nHost: x\r\n"
		for _, h := range headers {
			req += h + "\r\n"
		}
		io.WriteString(c, req+"\r\n")
		resp, err := http.ReadResponse(bufio.NewReader(c), &http.Request{Method: "CONNECT"})
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, c
	}

	if got, c := tlsConn(); got != 407 {
		c.Close()
		t.Errorf("no credentials over TLS: got %d, want 407", got)
	} else {
		c.Close()
	}
	good := "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("julio:pw"))
	if got, c := tlsConn(good); got != 200 {
		c.Close()
		t.Errorf("valid credentials over TLS: got %d, want 200", got)
	} else {
		c.Close()
	}
	bad := "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("julio:nope"))
	if got, c := tlsConn(bad); got != 407 {
		c.Close()
		t.Errorf("wrong password over TLS: got %d, want 407", got)
	} else {
		c.Close()
	}
}

// Bytes sent right after the CONNECT request (before the 200 is read) must reach
// the destination. They can already be in the server's read buffer when the
// connection is hijacked.
func TestBytesPipelinedAfterConnectAreNotLost(t *testing.T) {
	_, addr := echoServer(t, "tcp", "127.0.0.1:0")
	port := portOf(t, addr)
	srv := startProxy(t, config.ACL{Default: "permit"})

	for i := 0; i < 50; i++ {
		c, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		c.SetDeadline(time.Now().Add(5 * time.Second))
		payload := "pipelined-" + strconv.Itoa(i)
		io.WriteString(c, "CONNECT 127.0.0.1:"+port+" HTTP/1.1\r\nHost: x\r\n\r\n"+payload)
		br := bufio.NewReader(c)
		resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
		if err != nil || resp.StatusCode != 200 {
			c.Close()
			t.Fatalf("iteration %d: CONNECT failed: %v", i, err)
		}
		buf := make([]byte, len(payload))
		_, err = io.ReadFull(br, buf)
		c.Close()
		if err != nil || string(buf) != payload {
			t.Fatalf("iteration %d: pipelined bytes lost: got %q (%v), want %q", i, buf, err, payload)
		}
	}
}

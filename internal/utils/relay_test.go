package utils

import (
	"io"
	"net"
	"testing"
	"time"
)

// tcpPair returns two connected TCP endpoints: a (proxy side, client) and b
// (proxy side, destination), plus the peers that talk to them.
func tcpPair(t *testing.T) (proxyA, peerA, proxyB, peerB net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan net.Conn, 2)
	go func() {
		for i := 0; i < 2; i++ {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- c
		}
	}()
	peerA, _ = net.Dial("tcp", ln.Addr().String())
	peerB, _ = net.Dial("tcp", ln.Addr().String())
	proxyA = <-accepted
	proxyB = <-accepted
	t.Cleanup(func() {
		for _, c := range []net.Conn{proxyA, peerA, proxyB, peerB} {
			c.Close()
		}
	})
	return
}

func TestRelayHalfCloseKeepsReturnPath(t *testing.T) {
	proxyA, peerA, proxyB, peerB := tcpPair(t)
	done := make(chan struct{})
	go func() { Relay(proxyA, proxyB, 0, 0, 0); close(done) }()

	peerA.Write([]byte("request"))
	peerA.(*net.TCPConn).CloseWrite()
	// peerB sees the request, then EOF, and replies.
	got, _ := io.ReadAll(peerB)
	if string(got) != "request" {
		t.Fatalf("destination received %q", got)
	}
	peerB.Write([]byte("reply"))
	peerB.(*net.TCPConn).CloseWrite()
	back, _ := io.ReadAll(peerA)
	if string(back) != "reply" {
		t.Fatalf("client received %q after half-close", back)
	}
	<-done
}

func TestRelayIdleTimeoutClosesTunnel(t *testing.T) {
	proxyA, peerA, proxyB, peerB := tcpPair(t)
	done := make(chan struct{})
	go func() { Relay(proxyA, proxyB, 200*time.Millisecond, 0, 0); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("idle tunnel was not closed")
	}
	// The proxy closed its side: the client's read must fail, not wait.
	peerA.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := peerA.Read(make([]byte, 1)); err == nil {
		t.Error("client side should be closed after idle timeout")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Error("client was not closed; the read timed out")
	}
	_ = peerB
}

func TestRelayThrottlesBothDirections(t *testing.T) {
	proxyA, peerA, proxyB, peerB := tcpPair(t)
	go Relay(proxyA, proxyB, 0, 2, 0) // 2 KB/s each way

	data := make([]byte, 3*1024)
	start := time.Now()
	go func() { peerA.Write(data) }()
	n, _ := io.ReadFull(peerB, make([]byte, len(data)))
	elapsed := time.Since(start)
	if n != len(data) {
		t.Fatalf("read %d bytes", n)
	}
	// 3 KB at 2 KB/s needs about 1.5 s; anything much faster means no limit.
	if elapsed < 1*time.Second {
		t.Errorf("client->destination not throttled: %v for 3KB at 2KB/s", elapsed)
	}
}

func TestRelayMaxDurationClosesActiveTunnel(t *testing.T) {
	proxyA, peerA, proxyB, peerB := tcpPair(t)
	done := make(chan struct{})
	go func() { Relay(proxyA, proxyB, 0, 0, 300*time.Millisecond); close(done) }()

	// Keep traffic flowing so only the maximum duration can end the tunnel.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				peerA.Write([]byte("x"))
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()
	go io.Copy(io.Discard, peerB)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("active tunnel was not closed at max duration")
	}
	_ = proxyA
}

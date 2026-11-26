package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// buildBinary compiles the service once per test run.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "doppler")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// echoTarget accepts connections and echoes them back.
func echoTarget(t *testing.T) int {
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
			go func(c net.Conn) { defer c.Close(); io.Copy(c, c) }(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// lockedBuffer is written by the exec copier goroutine and read by the test.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// running is a started service and its combined log.
type running struct {
	cmd *exec.Cmd
	log *lockedBuffer
}

func start(t *testing.T, bin, cfgPath string) *running {
	t.Helper()
	log := &lockedBuffer{}
	cmd := exec.Command(bin, "-c", cfgPath)
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	return &running{cmd: cmd, log: log}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// connectStatus sends CONNECT through the service and returns the status line code.
func connectStatus(t *testing.T, port int, target string) int {
	t.Helper()
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		t.Fatalf("CONNECT %s: %v", target, err)
	}
	var code int
	fmt.Sscanf(line, "HTTP/1.1 %d", &code)
	return code
}

// waitFor polls until the log contains s at least n times, or times out. Counting
// matters: a repeated message (such as a second reload) must wait for its own copy.
func waitFor(t *testing.T, r *running, s string, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Count(r.log.String(), s) >= n {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("log did not contain %q %d time(s):\n%s", s, n, r.log.String())
}

// The service refuses to start with a bad config and lists every problem.
func TestStartupRejectsInvalidConfig(t *testing.T) {
	bin := buildBinary(t)
	cfg := filepath.Join(t.TempDir(), "config.yml")
	writeFile(t, cfg, "addr: \"5000\"\nACL:\n  default: allow\n  permit:\n    - 10.0.0.0/33\n")

	out, err := exec.Command(bin, "-c", cfg).CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v\n%s", err, out)
	}
	for _, want := range []string{"invalid configuration", "addr", "ACL.default", "10.0.0.0/33"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("startup message missing %q:\n%s", want, out)
		}
	}
}

// SIGHUP applies a new policy without a restart, keeps the running policy when
// the file is invalid, and SIGTERM closes open tunnels before exiting.
func TestReloadAndGracefulShutdown(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yml")
	port := freePort(t)
	target := echoTarget(t)
	dest := fmt.Sprintf("127.0.0.1:%d", target)

	writeFile(t, cfg, fmt.Sprintf("addr: \"127.0.0.1:%d\"\nlogger: false\naccess_log: false\nACL:\n  default: block\n  permit:\n    - 127.0.0.1\n", port))
	r := start(t, bin, cfg)
	waitFor(t, r, "Server listening", 1)

	if got := connectStatus(t, port, dest); got != 200 {
		t.Fatalf("permitted destination: %d, want 200", got)
	}

	// Revoke the permit and reload.
	writeFile(t, cfg, fmt.Sprintf("addr: \"127.0.0.1:%d\"\nlogger: false\naccess_log: false\nACL:\n  default: block\n", port))
	r.cmd.Process.Signal(syscall.SIGHUP)
	waitFor(t, r, "configuration reloaded", 1)
	if got := connectStatus(t, port, dest); got != 403 {
		t.Errorf("after reload without permit: %d, want 403", got)
	}

	// An invalid file must be rejected and the running policy kept.
	writeFile(t, cfg, fmt.Sprintf("addr: \"127.0.0.1:%d\"\nACL:\n  default: allow\n", port))
	r.cmd.Process.Signal(syscall.SIGHUP)
	waitFor(t, r, "reload failed", 1)
	if got := connectStatus(t, port, dest); got != 403 {
		t.Errorf("after rejected reload: %d, want 403 (policy must be unchanged)", got)
	}

	// Permit again, open a tunnel, then stop the service: the tunnel must end.
	writeFile(t, cfg, fmt.Sprintf("addr: \"127.0.0.1:%d\"\nlogger: false\naccess_log: false\nACL:\n  default: block\n  permit:\n    - 127.0.0.1\n", port))
	r.cmd.Process.Signal(syscall.SIGHUP)
	waitFor(t, r, "configuration reloaded", 2)

	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	br := bufio.NewReader(c)
	if line, _ := br.ReadString('\n'); !strings.Contains(line, "200") {
		t.Fatalf("tunnel not established: %q", line)
	}

	r.cmd.Process.Signal(syscall.SIGTERM)
	if _, err := io.ReadAll(br); err != nil && !errors.Is(err, io.EOF) {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			t.Fatal("open tunnel was not closed on shutdown")
		}
	}
	done := make(chan error, 1)
	go func() { done <- r.cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("exit after SIGTERM: %v\n%s", err, r.log.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("service did not exit after SIGTERM")
	}
}

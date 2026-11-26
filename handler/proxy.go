package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/helper"
	"github.com/juliotorresmoreno/doppler/internal/utils"
)

// Extra reasons for access events, beyond the ACL reasons in internal/utils.
const (
	reasonTooManyConnections = "too_many_connections"
	reasonTLSRequired        = "tls_required"
	reasonAuthMissing        = "auth_missing"
	reasonAuthBlocked        = "auth_blocked"
	reasonAuthFailed         = "auth_failed"
	reasonUpstreamError      = "upstream_error"
	reasonInternalError      = "internal_error"
	reasonShuttingDown       = "shutting_down"
	reasonUnsupported        = "unsupported_request"
)

type Proxy struct {
	Fallback http.Handler
	Logger   *utils.LoggerBD
	// slots limits concurrent proxied connections; nil means unlimited.
	slots     chan struct{}
	accessLog bool
}

func ProxyHandler(fallback http.Handler) http.Handler {
	conf, _ := config.GetConfig()
	p := &Proxy{
		Fallback:  fallback,
		Logger:    utils.NewLogger(),
		accessLog: conf.AccessLog,
	}
	if conf.MaxConnections > 0 {
		p.slots = make(chan struct{}, conf.MaxConnections)
	}
	return http.HandlerFunc(p.Handle)
}

// acquire takes a connection slot. It returns false when the limit is reached.
func (h *Proxy) acquire() bool {
	if h.slots != nil {
		select {
		case h.slots <- struct{}{}:
		default:
			return false
		}
	}
	utils.ActiveConnections.Add(1)
	return true
}

func (h *Proxy) release() {
	utils.ActiveConnections.Add(-1)
	if h.slots != nil {
		<-h.slots
	}
}

// access starts the event for a proxied request. Metrics are updated for every
// request; the JSON line is written only when access logging is on.
func (h *Proxy) access(event string, req *http.Request) *utils.AccessEvent {
	return utils.NewAccessEvent(event, clientIP(req), req.Method, req.Host)
}

func (h *Proxy) finish(ev *utils.AccessEvent) {
	utils.ObserveAccess(ev)
	if h.accessLog {
		ev.Emit()
	}
}

func (h *Proxy) Handle(res http.ResponseWriter, req *http.Request) {
	conf, err := config.GetConfig()
	if err != nil {
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}

	isProxyReq, protocol := utils.IsRequestToProxy(req)
	if !isProxyReq {
		h.Fallback.ServeHTTP(res, req)
		return
	}

	event := protocol
	if protocol != "http" && protocol != "connect" {
		event = "unknown"
	}
	ev := h.access(event, req)
	defer h.finish(ev)

	if !h.acquire() {
		if ev != nil {
			ev.SetDecision(false, reasonTooManyConnections, nil)
		}
		http.Error(res, "Too Many Connections", http.StatusServiceUnavailable)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusServiceUnavailable)
		return
	}
	defer h.release()

	if conf.Auth.Enabled {
		// Credentials are only accepted over the TLS listener; on plain HTTP the
		// request is refused before its Proxy-Authorization header is used.
		if req.TLS == nil {
			if ev != nil {
				ev.SetDecision(false, reasonTLSRequired, nil)
			}
			http.Error(res, "Proxy authentication requires the TLS listener", http.StatusForbidden)
			h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
			return
		}
		user, err := h.BasicAuth(req.Header.Get("Proxy-Authorization"), clientIP(req))
		if err != nil {
			if ev != nil {
				ev.SetDecision(false, authReason(err), nil)
			}
			h.AuthRequired(res, req)
			h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusProxyAuthRequired)
			return
		}
		if ev != nil {
			ev.User = user
		}
		req.Header.Del("Proxy-Authorization")
		req.Header.Del("Proxy-Connection")
	}

	switch protocol {
	case "http":
		h.HandleHTTP(res, req, ev)
	case "connect":
		h.HandleConnect(res, req, ev)
	default:
		if ev != nil {
			ev.SetDecision(false, reasonUnsupported, nil)
		}
		http.Error(res, "Unsupported protocol", http.StatusBadRequest)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusBadRequest)
	}
}

// authReason maps an authentication error to an access-log reason.
func authReason(err error) string {
	switch {
	case errors.Is(err, errAuthMissing):
		return reasonAuthMissing
	case errors.Is(err, errAuthBlocked):
		return reasonAuthBlocked
	default:
		return reasonAuthFailed
	}
}

// resolvedStrings returns the IPs as text for the access log.
func resolvedStrings(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

// countingReader counts the bytes read from a request body.
type countingReader struct {
	io.ReadCloser
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	c.n += int64(n)
	return n, err
}

func (h *Proxy) HandleHTTP(w http.ResponseWriter, req *http.Request, ev *utils.AccessEvent) {
	host := req.URL.Hostname()
	conf, err := config.GetConfig()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}

	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	d := utils.Decide(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default, conf.ACL.BlockSpecialRanges)
	if ev != nil {
		ev.SetDecision(d.Allowed, d.Reason, resolvedStrings(d.IPs))
	}
	if !d.Allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		if ev != nil {
			ev.Status = http.StatusForbidden
		}
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
		return
	}

	if utils.IsWebSocket(req) {
		h.handleWebSocket(w, req, d.IPs, ev)
		return
	}

	// Each forwarded request gets its own connection to the destination (like a
	// CONNECT tunnel). Close idle connections when the request ends, so none are
	// left open waiting for the transport's idle timeout.
	tr := h.transport()
	defer tr.CloseIdleConnections()
	tr.DialContext = dialApproved(d.IPs)
	body := &countingReader{ReadCloser: req.Body}
	req.Body = body
	resp, err := tr.RoundTrip(req)
	if ev != nil {
		ev.BytesUp = body.n
	}
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		if ev != nil {
			ev.Reason = reasonUpstreamError
			ev.Status = http.StatusServiceUnavailable
		}
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	utils.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	n, _ := io.Copy(w, resp.Body)
	if ev != nil {
		ev.Status = resp.StatusCode
		ev.BytesDown = n
	}

	h.Logger.Register(req.Host, req.Method, resp.StatusCode)
}

func (h *Proxy) handleWebSocket(w http.ResponseWriter, req *http.Request, ips []net.IP, ev *utils.AccessEvent) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}

	rawConn, rw, err := hj.Hijack()
	if err != nil {
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}
	clientConn := helper.NewBufferedConn(rawConn, rw.Reader)

	hostPort := req.Host
	if !strings.Contains(hostPort, ":") {
		hostPort += ":80"
	}

	targetConn, err := dialApprovedAddr(ips, hostPort)
	if err != nil {
		clientConn.Close()
		if ev != nil {
			ev.Reason = reasonUpstreamError
		}
		h.Logger.RegisterWithLevel(utils.ERROR, hostPort, "WEBSOCKET", 0)
		return
	}

	release, ok := tunnels.track(clientConn, targetConn)
	if !ok {
		clientConn.Close()
		targetConn.Close()
		if ev != nil {
			ev.Reason = reasonShuttingDown
		}
		return
	}
	defer release()

	if err := req.Write(targetConn); err != nil {
		clientConn.Close()
		targetConn.Close()
		if ev != nil {
			ev.Reason = reasonUpstreamError
		}
		h.Logger.RegisterWithLevel(utils.ERROR, hostPort, "WEBSOCKET", 0)
		return
	}
	conf, _ := config.GetConfig()
	if ev != nil {
		ev.Status = http.StatusSwitchingProtocols
	}
	h.Logger.RegisterWithLevel(utils.INFO, hostPort, "WEBSOCKET", 0)
	up, down := utils.Relay(clientConn, targetConn, idleTimeout(conf), conf.Limit, maxTunnel(conf))
	if ev != nil {
		ev.BytesUp, ev.BytesDown = up, down
	}
}

func (h *Proxy) transport() *http.Transport {
	return &http.Transport{
		Proxy:              nil,
		DisableCompression: false,
		ForceAttemptHTTP2:  true,
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
	}
}

func (h *Proxy) HandleConnect(w http.ResponseWriter, req *http.Request, ev *utils.AccessEvent) {
	conf, err := config.GetConfig()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	host := req.URL.Hostname()
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	d := utils.Decide(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default, conf.ACL.BlockSpecialRanges)
	if ev != nil {
		ev.SetDecision(d.Allowed, d.Reason, resolvedStrings(d.IPs))
	}
	if !d.Allowed {
		http.Error(w, "Forbidden", http.StatusForbidden)
		if ev != nil {
			ev.Status = http.StatusForbidden
		}
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, "CONNECT", http.StatusForbidden)
		return
	}

	destConn, err := dialApprovedAddr(d.IPs, req.Host)
	if err != nil {
		// The detail stays in the log; the client only learns it could not connect.
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		if ev != nil {
			ev.Reason = reasonUpstreamError
			ev.Status = http.StatusServiceUnavailable
		}
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	clientConn, err := helper.GetHijack(w)
	if err != nil {
		destConn.Close()
		if ev != nil {
			ev.Reason = reasonInternalError
			ev.Status = http.StatusInternalServerError
		}
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	release, ok := tunnels.track(clientConn, destConn)
	if !ok {
		clientConn.Close()
		destConn.Close()
		if ev != nil {
			ev.Reason = reasonShuttingDown
		}
		return
	}
	defer release()

	if ev != nil {
		ev.Status = http.StatusOK
	}
	h.Logger.RegisterWithLevel(utils.INFO, req.Host, "CONNECT", http.StatusOK)
	up, down := utils.Relay(clientConn, destConn, idleTimeout(conf), conf.Limit, maxTunnel(conf)) // limit in KB/s
	if ev != nil {
		ev.BytesUp, ev.BytesDown = up, down
	}
}

// idleTimeout converts the configured idle_timeout (seconds) to a duration.
func idleTimeout(conf config.Config) time.Duration {
	return time.Duration(conf.IdleTimeout) * time.Second
}

// maxTunnel converts the configured max_tunnel_seconds to a duration.
func maxTunnel(conf config.Config) time.Duration {
	return time.Duration(conf.MaxTunnelSeconds) * time.Second
}

// dialApproved returns a DialContext that connects only to the addresses the
// ACL approved, ignoring the name the transport asks for.
func dialApproved(ips []net.IP) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		return dialIPs(ctx, network, ips, port)
	}
}

// dialApprovedAddr dials hostPort ("host:port") using only the approved addresses.
func dialApprovedAddr(ips []net.IP, hostPort string) (net.Conn, error) {
	_, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return dialIPs(ctx, "tcp", ips, port)
}

func dialIPs(ctx context.Context, network string, ips []net.IP, port string) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, errors.New("no approved address to connect to")
	}
	var d net.Dialer
	var lastErr error
	for _, ip := range ips {
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

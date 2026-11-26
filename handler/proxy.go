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

type Proxy struct {
	Fallback http.Handler
	Logger   *utils.LoggerBD
	// slots limits concurrent proxied connections; nil means unlimited.
	slots chan struct{}
}

func ProxyHandler(fallback http.Handler) http.Handler {
	conf, _ := config.GetConfig()
	p := &Proxy{
		Fallback: fallback,
		Logger:   utils.NewLogger(),
	}
	if conf.MaxConnections > 0 {
		p.slots = make(chan struct{}, conf.MaxConnections)
	}
	return http.HandlerFunc(p.Handle)
}

// acquire takes a connection slot. It returns false when the limit is reached.
func (h *Proxy) acquire() bool {
	if h.slots == nil {
		return true
	}
	select {
	case h.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (h *Proxy) release() {
	if h.slots != nil {
		<-h.slots
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

	if !h.acquire() {
		http.Error(res, "Too Many Connections", http.StatusServiceUnavailable)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusServiceUnavailable)
		return
	}
	defer h.release()

	if conf.Auth.Enabled {
		// Credentials are only accepted over the TLS listener; on plain HTTP the
		// request is refused before its Proxy-Authorization header is used.
		if req.TLS == nil {
			http.Error(res, "Proxy authentication requires the TLS listener", http.StatusForbidden)
			h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
			return
		}
		authHeader := req.Header.Get("Proxy-Authorization")
		if err := h.BasicAuth(authHeader, clientIP(req)); err != nil {
			h.AuthRequired(res, req)
			h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusProxyAuthRequired)
			return
		}
		req.Header.Del("Proxy-Authorization")
		req.Header.Del("Proxy-Connection")
	}

	switch protocol {
	case "http":
		h.HandleHTTP(res, req)
	case "connect":
		h.HandleConnect(res, req)
	default:
		http.Error(res, "Unsupported protocol", http.StatusBadRequest)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusBadRequest)
	}
}

func (h *Proxy) HandleHTTP(w http.ResponseWriter, req *http.Request) {
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

	ips, ok := utils.Authorize(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default, conf.ACL.BlockSpecialRanges)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
		return
	}

	if utils.IsWebSocket(req) {
		h.handleWebSocket(w, req, ips)
		return
	}

	// Each forwarded request gets its own connection to the destination (like a
	// CONNECT tunnel). Close idle connections when the request ends, so none are
	// left open waiting for the transport's idle timeout.
	tr := h.transport()
	defer tr.CloseIdleConnections()
	tr.DialContext = dialApproved(ips)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	utils.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	h.Logger.Register(req.Host, req.Method, resp.StatusCode)
}

func (h *Proxy) handleWebSocket(w http.ResponseWriter, req *http.Request, ips []net.IP) {
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
		h.Logger.RegisterWithLevel(utils.ERROR, hostPort, "WEBSOCKET", 0)
		return
	}

	release, ok := tunnels.track(clientConn, targetConn)
	if !ok {
		clientConn.Close()
		targetConn.Close()
		return
	}
	defer release()

	if err := req.Write(targetConn); err != nil {
		clientConn.Close()
		targetConn.Close()
		h.Logger.RegisterWithLevel(utils.ERROR, hostPort, "WEBSOCKET", 0)
		return
	}
	conf, _ := config.GetConfig()
	h.Logger.RegisterWithLevel(utils.INFO, hostPort, "WEBSOCKET", 0)
	utils.Relay(clientConn, targetConn, idleTimeout(conf), conf.Limit, maxTunnel(conf))
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

func (h *Proxy) HandleConnect(w http.ResponseWriter, req *http.Request) {
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

	ips, ok := utils.Authorize(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default, conf.ACL.BlockSpecialRanges)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, "CONNECT", http.StatusForbidden)
		return
	}

	destConn, err := dialApprovedAddr(ips, req.Host)
	if err != nil {
		// The detail stays in the log; the client only learns it could not connect.
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	clientConn, err := helper.GetHijack(w)
	if err != nil {
		destConn.Close()
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	release, ok := tunnels.track(clientConn, destConn)
	if !ok {
		clientConn.Close()
		destConn.Close()
		return
	}
	defer release()

	h.Logger.RegisterWithLevel(utils.INFO, req.Host, "CONNECT", http.StatusOK)
	utils.Relay(clientConn, destConn, idleTimeout(conf), conf.Limit, maxTunnel(conf)) // limit in KB/s
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

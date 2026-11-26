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
}

func ProxyHandler(fallback http.Handler) http.Handler {
	p := &Proxy{
		Fallback: fallback,
		Logger:   utils.NewLogger(),
	}
	return http.HandlerFunc(p.Handle)
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

	tr := h.transport()
	tr.DialContext = dialApproved(ips)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
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

	req.Write(targetConn)
	go io.Copy(targetConn, clientConn)
	go io.Copy(clientConn, targetConn)

	h.Logger.RegisterWithLevel(utils.INFO, hostPort, "WEBSOCKET", 0)
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
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	clientConn, err := helper.GetHijack(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	go io.Copy(destConn, clientConn)
	if conf.Limit > 0 {
		go utils.CopyWithRateLimit(clientConn, destConn, conf.Limit) // limit in KB/s
	} else {
		go io.Copy(clientConn, destConn)
	}
	h.Logger.RegisterWithLevel(utils.INFO, req.Host, "CONNECT", http.StatusOK)
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

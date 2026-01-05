package handler

import (
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
		authHeader := req.Header.Get("Proxy-Authorization")
		if err := h.BasicAuth(authHeader); err != nil {
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

	if !utils.IsAllowedHost(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
		return
	}

	if utils.IsWebSocket(req) {
		h.handleWebSocket(w, req)
		return
	}

	resp, err := h.transport().RoundTrip(req)
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

func (h *Proxy) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		h.Logger.RegisterWithLevel(utils.ERROR, req.Host, req.Method, http.StatusInternalServerError)
		return
	}

	hostPort := req.Host
	if !strings.Contains(hostPort, ":") {
		hostPort += ":80"
	}

	targetConn, err := net.Dial("tcp", hostPort)
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

	if !utils.IsAllowedHost(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, "CONNECT", http.StatusForbidden)
		return
	}

	destConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
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

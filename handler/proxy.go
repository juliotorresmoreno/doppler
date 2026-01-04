package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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

	if !isAllowedHost(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		h.Logger.RegisterWithLevel(utils.WARN, req.Host, req.Method, http.StatusForbidden)
		return
	}

	if isWebSocket(req) {
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

	copyHeader(w.Header(), resp.Header)
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

func getIpAddress(host string) []net.IP {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil
	}
	return ips
}

func isAllowedHost(host string, permitList, blockList []string, defaultPolicy string) bool {
	hosts := make([]string, 0)
	hosts = append(hosts, host)

	if ip := getIpAddress(host); ip != nil {
		for _, ip := range ip {
			hosts = append(hosts, ip.String())
		}
	}

	for _, host := range hosts {
		for _, blocked := range blockList {
			if strings.EqualFold(host, blocked) {
				return false
			}
		}

		for _, permitted := range permitList {
			if strings.EqualFold(host, permitted) {
				return true
			}
		}
	}

	return strings.EqualFold(defaultPolicy, "permit")
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
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

	if !isAllowedHost(host, conf.ACL.Permit, conf.ACL.Block, conf.ACL.Default) {
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

func (h *Proxy) AuthRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Proxy-Authenticate", `Basic realm="Doppler"`)
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte(authenticationRequiredHTML))
}

const authenticationRequiredHTML = `<html>
<head><title>Proxy Authentication Required</title></head>
<body>
<h1>Proxy Authentication Required</h1>
<p>This proxy server requires authentication to access the requested resource.</p>
</body>
</html>`

func (h *Proxy) BasicAuth(credentials string) error {
	conf, err := config.GetConfig()
	if err != nil {
		return errors.New("Unauthorized")
	}

	if len(credentials) < 6 {
		return errors.New("Unauthorized")
	}

	decoded, err := base64.StdEncoding.DecodeString(credentials[6:])
	if err != nil {
		return errors.New("Unauthorized")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return errors.New("Unauthorized")
	}

	username, password := parts[0], parts[1]

	if storedHash, ok := conf.Auth.Users[username]; ok {
		sum := sha256.Sum256([]byte(strings.Trim(password, " ")))
		hashStr := fmt.Sprintf("%x", sum)
		if hashStr == storedHash {
			return nil
		}
	}

	return errors.New("Unauthorized")
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

package handler

import (
	"encoding/base64"
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
	Logger   utils.Logger
}

func ProxyHandler(fallback http.Handler) http.Handler {
	p := &Proxy{
		Fallback: fallback,
		Logger:   &utils.LoggerBD{},
	}

	return http.HandlerFunc(p.Handle)
}

func (h *Proxy) Handle(res http.ResponseWriter, req *http.Request) {
	config, err := config.GetConfig()
	if err != nil {
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	isRequestToProxy, protocol := utils.IsRequestToProxy(req)
	if !isRequestToProxy {
		h.Fallback.ServeHTTP(res, req)
		return
	}

	// Authentication
	if config.Auth.Enabled {
		authHeader := req.Header.Get("Proxy-Authorization")
		err := h.BasicAuth(authHeader)
		if err != nil {
			h.AuthRequired(res, req)
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
	}
}

func (h *Proxy) HandleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)

	h.Logger.Register(req.Host, req.Method, resp.StatusCode)
}

func (h *Proxy) HandleConnect(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	clientConn, err := helper.GetHijack(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.Logger.Register(r.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	go io.Copy(destConn, clientConn)
	//go utils.CopyWithRateLimit(clientConn, destConn, 5000)
	go io.Copy(clientConn, destConn)

	h.Logger.Register(r.Host, "CONNECT", http.StatusOK)
}

const authenticationRequiredHTML = `
<!DOCTYPE HTML "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>407 Proxy Authentication Required</title>
</head><body>
<h1>Proxy Authentication Required</h1>
<p>This server could not verify that you
are authorized to access the document
requested.  Either you supplied the wrong
credentials (e.g., bad password), or your
browser doesn't understand how to supply
the credentials required.</p>
</body></html>
`

const ACLDeniedHTML = `
<!DOCTYPE HTML "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>401 Proxy Authentication Denied</title>
</head><body>
<h1>Proxy Authentication Denied</h1>
<p>You do not have permission to access this site.</p>
</body></html>
`

func (h *Proxy) AuthRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"Doppler\"")
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte(authenticationRequiredHTML))
}

// basicAuth .
func (h *Proxy) BasicAuth(credentials string) error {
	config, err := config.GetConfig()
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
	splitData := strings.Split(string(decoded), ":")
	if len(splitData) == 1 {
		return errors.New("Unauthorized")
	}
	username := splitData[0]
	password := splitData[1]

	// TODO: Validate username and password
	if user, ok := config.Auth.Users[username]; ok && user == password {
		return nil
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

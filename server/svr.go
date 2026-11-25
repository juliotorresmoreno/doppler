package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/middleware"
)

// Configure returns the plain HTTP server and, when tls_addr is set, a second
// server for TLS connections that shares the same handler. The TLS server is
// nil when TLS is disabled.
func Configure() (*http.Server, *http.Server) {
	r := mux.NewRouter()
	r.Use(middleware.Cors)

	conf, _ := config.GetConfig()
	h := handler.ProxyHandler(r)

	server := &http.Server{
		Addr:         conf.Addr,
		Handler:      h,
		ReadTimeout:  time.Duration(conf.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(conf.WriteTimeout) * time.Second,
	}

	var tlsServer *http.Server
	if conf.TLSAddr != "" {
		tlsServer = &http.Server{
			Addr:         conf.TLSAddr,
			Handler:      h,
			ReadTimeout:  time.Duration(conf.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(conf.WriteTimeout) * time.Second,
			// A non-nil empty map disables automatic HTTP/2 (ALPN "h2").
			// CONNECT tunnels need HTTP/1.1 hijacking, which HTTP/2 does not support.
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		}
	}
	return server, tlsServer
}

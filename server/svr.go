package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/internal/utils"
	"github.com/juliotorresmoreno/doppler/middleware"
)

// Configure returns the plain HTTP server and, when tls_addr is set, a second
// server for TLS connections that shares the same handler. The TLS server is
// nil when TLS is disabled.
func Configure() (*http.Server, *http.Server) {
	r := mux.NewRouter()
	r.Use(middleware.Cors)
	r.HandleFunc("/proxy.pac", servePAC).Methods("GET")

	conf, _ := config.GetConfig()
	h := handler.ProxyHandler(r)

	server := &http.Server{
		Addr:           conf.Addr,
		MaxHeaderBytes: maxHeaderBytes(conf),
		IdleTimeout:    time.Duration(conf.IdleTimeout) * time.Second,
		Handler:        h,
		ReadTimeout:    time.Duration(conf.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(conf.WriteTimeout) * time.Second,
	}

	var tlsServer *http.Server
	if conf.TLSAddr != "" {
		tlsServer = &http.Server{
			Addr:           conf.TLSAddr,
			MaxHeaderBytes: maxHeaderBytes(conf),
			IdleTimeout:    time.Duration(conf.IdleTimeout) * time.Second,
			Handler:        h,
			ReadTimeout:    time.Duration(conf.ReadTimeout) * time.Second,
			WriteTimeout:   time.Duration(conf.WriteTimeout) * time.Second,
			// A non-nil empty map disables automatic HTTP/2 (ALPN "h2").
			// CONNECT tunnels need HTTP/1.1 hijacking, which HTTP/2 does not support.
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		}
	}
	return server, tlsServer
}

func maxHeaderBytes(conf config.Config) int {
	if conf.MaxHeaderBytes > 0 {
		return conf.MaxHeaderBytes
	}
	return http.DefaultMaxHeaderBytes
}

// ConfigureMetrics returns the HTTP server that exposes /metrics on its own
// address, or nil when metrics_addr is empty.
func ConfigureMetrics() *http.Server {
	conf, _ := config.GetConfig()
	if conf.MetricsAddr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", utils.MetricsHandler())
	return &http.Server{
		Addr:         conf.MetricsAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

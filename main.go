package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/server"
)

// shutdownGrace is how long in-flight requests get to finish after a stop signal.
const shutdownGrace = 5 * time.Second

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		// Refuse to start with a configuration that did not load: requests would
		// otherwise fail or run with partial settings.
		log.Fatal("invalid configuration: ", err)
	}
	svr, tlsSvr := server.Configure()
	metricsSvr := server.ConfigureMetrics()

	if tlsSvr != nil {
		if conf.TLSCert == "" || conf.TLSKey == "" {
			log.Fatal("tls_addr requires both tls_cert and tls_key")
		}
		// Validate the certificate pair before starting any listener.
		if _, err := tls.LoadX509KeyPair(conf.TLSCert, conf.TLSKey); err != nil {
			log.Fatal("invalid TLS certificate/key: ", err)
		}

		tlsConn, err := net.Listen("tcp", tlsSvr.Addr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("TLS server listening on " + tlsSvr.Addr)
		go func() {
			if err := tlsSvr.ServeTLS(tlsConn, conf.TLSCert, conf.TLSKey); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
	}

	conn, err := net.Listen("tcp", svr.Addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server listening on " + svr.Addr)
	go func() {
		if err := svr.Serve(conn); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	if metricsSvr != nil {
		metricsConn, err := net.Listen("tcp", metricsSvr.Addr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Metrics listening on " + metricsSvr.Addr)
		go func() {
			if err := metricsSvr.Serve(metricsConn); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
	}

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	<-stop.Done()
	log.Println("shutting down: closing tunnels")

	// Tunnels are hijacked, so the HTTP shutdown does not wait for them; close them first.
	handler.CloseTunnels()
	ctx, done := context.WithTimeout(context.Background(), shutdownGrace)
	defer done()
	if metricsSvr != nil {
		metricsSvr.Shutdown(ctx)
	}
	if tlsSvr != nil {
		tlsSvr.Shutdown(ctx)
	}
	if err := svr.Shutdown(ctx); err != nil {
		log.Println("shutdown:", err)
	}
}

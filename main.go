package main

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/server"
)

func main() {
	conf, _ := config.GetConfig()
	svr, tlsSvr := server.Configure()

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
			log.Fatal(tlsSvr.ServeTLS(tlsConn, conf.TLSCert, conf.TLSKey))
		}()
	}

	conn, err := net.Listen("tcp", svr.Addr)
	log.Println("Server listening on " + svr.Addr)

	if err != nil {
		log.Fatal(err)
	}

	err = svr.Serve(conn)
	if err != nil {
		log.Fatal(err)
	}
}

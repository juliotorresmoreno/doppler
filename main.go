package main

import (
	"log"
	"net"

	"github.com/juliotorresmoreno/doppler/server"
)

func main() {
	svr := server.Configure()
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

package main

import (
	"flag"
	"log"

	"github.com/juliotorresmoreno/doppler/expose/client"
)

var alias = flag.String("alias", "development", "alias to expose service")
var expose = flag.String("expose", "localhost:5000", "http service to expose")
var protocol = flag.String("protocol", "http", "http service protocol")
var addr = flag.String("addr", "localhost:4080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	client.Reverse(*addr, *alias, *expose, *protocol)
}

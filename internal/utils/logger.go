package utils

import (
	"log"

	"github.com/juliotorresmoreno/doppler/config"
)

type Logger interface {
	Register(url, method string, status int)
}

type LoggerBD struct {
}

func (el *LoggerBD) Register(url, method string, status int) {
	conf, _ := config.GetConfig()
	if !conf.Logger {
		return
	}
	log.Println(method, url, status)
}

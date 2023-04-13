package utils

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/juliotorresmoreno/doppler/config"
)

type Logger interface {
	Register(method string, status int, header http.Header, body *bytes.Buffer)
}

type LoggerBD struct {
}

func (el *LoggerBD) Register(method string, status int, header http.Header, body *bytes.Buffer) {
	var content = ""
	if body != nil {
		content = body.String()
	}
	conf, _ := config.GetConfig()
	if !conf.Logger {
		return
	}
	fmt.Println(method, status, header, content)
}

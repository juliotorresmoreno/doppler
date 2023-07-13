package utils

import (
	"net/http"
	"strings"
)

func IsRequestToProxy(req *http.Request) (bool, string) {
	if req.Method == "CONNECT" {
		return true, "connect"
	}
	if strings.HasPrefix(req.URL.String(), "http") {
		return true, "http"
	}
	path := strings.Split(req.URL.Path, "?")[0]
	connection := req.Header.Get("Connection")
	if path == "/echo" && connection == "Upgrade" {
		return true, "websocket"
	}
	return false, ""
}

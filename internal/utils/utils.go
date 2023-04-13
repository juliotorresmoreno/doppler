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
	return false, ""
}

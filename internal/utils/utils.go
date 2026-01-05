package utils

import (
	"net"
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

func IsWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func GetIpAddress(host string) []net.IP {
	ips, err := net.LookupIP(host)
	if err != nil {
		return make([]net.IP, 0)
	}
	return ips
}

func IsAllowedHost(host string, permitList, blockList []string, defaultPolicy string) bool {
	hosts := make([]string, 0)
	hosts = append(hosts, host)

	for _, ip := range GetIpAddress(host) {
		hosts = append(hosts, ip.String())
	}

	for _, host := range hosts {
		for _, blocked := range blockList {
			if strings.EqualFold(host, blocked) {
				return false
			}
		}

		for _, permitted := range permitList {
			if strings.EqualFold(host, permitted) {
				return true
			}
		}
	}

	return strings.EqualFold(defaultPolicy, "permit")
}

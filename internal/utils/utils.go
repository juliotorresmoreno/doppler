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

// GetIpAddress resolves host. The error is returned so callers can tell a
// failed lookup apart from a host with no addresses.
func GetIpAddress(host string) ([]net.IP, error) {
	return LookupIP(host)
}

// LookupIP resolves a host name. It is a variable so tests can supply fixed
// answers instead of depending on the machine's DNS.
var LookupIP = net.LookupIP

// Authorize decides whether host may be reached. It resolves the name exactly
// once and returns the addresses the caller must connect to; connecting to the
// name again would let DNS answer differently and bypass the IP rules.
//
// Precedence (see README, "Policy evaluation"):
//  1. A name on the block list, or any resolved address on the block list
//     (single IP or CIDR), denies.
//  2. A name on the permit list allows.
//  3. A name that cannot be resolved is denied (fail closed).
//  4. If blockSpecial is set, any resolved address in a special-purpose range
//     (loopback, private, link-local, reserved...) denies unless that address
//     is explicitly permitted by IP or CIDR.
//  5. A resolved address on the permit list allows.
//  6. Anything else follows defaultPolicy.
func Authorize(host string, permitList, blockList []string, defaultPolicy string, blockSpecial bool) ([]net.IP, bool) {
	if matchName(host, blockList) {
		return nil, false
	}

	ips, err := GetIpAddress(host)
	if err != nil {
		if matchName(host, permitList) {
			return nil, true // allowed by name; the dial will fail and report it
		}
		return nil, false
	}

	for _, ip := range ips {
		if matchIP(ip, blockList) {
			return nil, false
		}
	}
	if matchName(host, permitList) {
		return ips, true
	}
	if blockSpecial {
		for _, ip := range ips {
			if isSpecial(ip) && !matchIP(ip, permitList) {
				return nil, false
			}
		}
	}
	for _, ip := range ips {
		if matchIP(ip, permitList) {
			return ips, true
		}
	}

	return ips, strings.EqualFold(defaultPolicy, "permit")
}

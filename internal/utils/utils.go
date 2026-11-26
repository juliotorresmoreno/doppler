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

// Reasons reported in access logs and decisions.
const (
	ReasonBlockName     = "block_name"
	ReasonBlockAddress  = "block_address"
	ReasonPermitName    = "permit_name"
	ReasonPermitAddress = "permit_address"
	ReasonSpecialRange  = "special_range"
	ReasonDNSFailure    = "dns_failure"
	ReasonDefaultPermit = "default_permit"
	ReasonDefaultBlock  = "default_block"
)

// Decision is the outcome of evaluating a destination.
type Decision struct {
	Allowed bool
	Reason  string
	// IPs are the resolved addresses; the connection must go only to these.
	IPs []net.IP
}

// Decide evaluates host against the ACL. It resolves the name exactly once and
// returns the addresses the caller must connect to; connecting to the name again
// would let DNS answer differently and bypass the IP rules.
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
func Decide(host string, permitList, blockList []string, defaultPolicy string, blockSpecial bool) Decision {
	if matchName(host, blockList) {
		return Decision{Reason: ReasonBlockName}
	}

	ips, err := GetIpAddress(host)
	if err != nil {
		if matchName(host, permitList) {
			// Allowed by name; the dial will fail and report it.
			return Decision{Allowed: true, Reason: ReasonPermitName}
		}
		return Decision{Reason: ReasonDNSFailure}
	}

	for _, ip := range ips {
		if matchIP(ip, blockList) {
			return Decision{Reason: ReasonBlockAddress, IPs: ips}
		}
	}
	if matchName(host, permitList) {
		return Decision{Allowed: true, Reason: ReasonPermitName, IPs: ips}
	}
	if blockSpecial {
		for _, ip := range ips {
			if isSpecial(ip) && !matchIP(ip, permitList) {
				return Decision{Reason: ReasonSpecialRange, IPs: ips}
			}
		}
	}
	for _, ip := range ips {
		if matchIP(ip, permitList) {
			return Decision{Allowed: true, Reason: ReasonPermitAddress, IPs: ips}
		}
	}

	if strings.EqualFold(defaultPolicy, "permit") {
		return Decision{Allowed: true, Reason: ReasonDefaultPermit, IPs: ips}
	}
	return Decision{Reason: ReasonDefaultBlock, IPs: ips}
}

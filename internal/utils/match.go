package utils

import (
	"net"
	"strings"
)

// matchName reports whether host matches an entry of a name list.
//
//   - "example.com" matches exactly example.com (not its subdomains).
//   - "*.example.com" matches any subdomain of example.com at any depth
//     (a.example.com, a.b.example.com) but not example.com itself.
//
// Matching is case-insensitive and ignores a trailing dot.
func matchName(host string, list []string) bool {
	host = normalizeName(host)
	for _, entry := range list {
		e := normalizeName(entry)
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, "*.") {
			suffix := e[1:] // ".example.com"
			if len(host) > len(suffix) && strings.HasSuffix(host, suffix) {
				return true
			}
			continue
		}
		if host == e {
			return true
		}
	}
	return false
}

// matchIP reports whether ip matches an entry of an address list. Entries may be
// single addresses ("10.1.2.3", "2001:db8::1") or CIDR ranges ("10.0.0.0/8").
func matchIP(ip net.IP, list []string) bool {
	for _, entry := range list {
		e := strings.TrimSpace(entry)
		if _, network, err := net.ParseCIDR(e); err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}
		if single := net.ParseIP(e); single != nil && single.Equal(ip) {
			return true
		}
	}
	return false
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
}

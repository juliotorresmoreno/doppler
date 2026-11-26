package utils

import (
	"net"
	"strings"

	"golang.org/x/net/idna"
)

// matchName reports whether host matches an entry of a name list.
//
//   - "example.com" matches exactly example.com (not its subdomains).
//   - "*.example.com" matches any subdomain of example.com at any depth
//     (a.example.com, a.b.example.com) but not example.com itself.
//
// Matching is case-insensitive, ignores a trailing dot, and compares
// internationalized names in punycode form.
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

// normalizeName puts a host name in one comparable form: ASCII, lower case, no
// trailing dot. Internationalized names are converted to their punycode form,
// so "MÜNCHEN.de" and "xn--mnchen-3ya.de" compare equal. A name that cannot be
// converted is kept as-is and will simply not match.
func normalizeName(s string) string {
	s = strings.TrimSuffix(strings.TrimSpace(s), ".")
	// The wildcard label is not a valid IDNA label, so convert what follows it.
	if strings.HasPrefix(s, "*.") {
		return "*." + normalizeName(s[2:])
	}
	if ascii, err := idna.Lookup.ToASCII(s); err == nil {
		s = ascii
	}
	return strings.ToLower(s)
}

// specialRanges are reserved or non-public networks that the net package's
// predicates do not cover (shared address space, documentation, benchmarking, etc.).
var specialRanges = mustParseCIDRs(
	"0.0.0.0/8",       // "this" network
	"100.64.0.0/10",   // carrier-grade NAT
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // documentation
	"198.18.0.0/15",   // benchmarking
	"198.51.100.0/24", // documentation
	"203.0.113.0/24",  // documentation
	"240.0.0.0/4",     // reserved
	"2001:db8::/32",   // documentation
	"100::/64",        // discard
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

// isSpecial reports whether ip is loopback, private, link-local, multicast,
// unspecified or in one of the reserved ranges above.
func isSpecial(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range specialRanges {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

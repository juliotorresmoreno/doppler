package utils

import (
	"net"
	"testing"
)

// FuzzMatchName: name matching must not panic for any host or list entry.
func FuzzMatchName(f *testing.F) {
	f.Add("a.example.com", "*.example.com")
	f.Add("EXAMPLE.com.", "example.com")
	f.Add("xn--mnchen-3ya.de", "münchen.de")
	f.Add("", "*.")
	f.Fuzz(func(t *testing.T, host, entry string) {
		_ = matchName(host, []string{entry})
	})
}

// FuzzMatchIP: CIDR and address matching must not panic on any entry.
func FuzzMatchIP(f *testing.F) {
	f.Add("127.0.0.1", "127.0.0.0/8")
	f.Add("::1", "::1/128")
	f.Add("10.0.0.1", "10.0.0.0/33")
	f.Fuzz(func(t *testing.T, ip, entry string) {
		_ = matchIP(parseIPOrNil(ip), []string{entry})
	})
}

func parseIPOrNil(s string) net.IP {
	return net.ParseIP(s)
}

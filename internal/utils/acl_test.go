package utils

import "testing"

const unresolvable = "doppler-nonexistent-host.invalid"

func allowed(host string, permit, block []string, def string) bool {
	d := Decide(host, permit, block, def, false)
	return d.Allowed
}

func TestDNSFailureFailsClosedWithPermitDefault(t *testing.T) {
	if allowed(unresolvable, nil, nil, "permit") {
		t.Fatal("unresolvable host must be denied even when default is permit")
	}
}

func TestDNSFailureStillAllowsExplicitPermit(t *testing.T) {
	if !allowed(unresolvable, []string{unresolvable}, nil, "block") {
		t.Fatal("explicitly permitted name should be allowed")
	}
}

func TestBlockListWinsOverPermit(t *testing.T) {
	if allowed(unresolvable, []string{unresolvable}, []string{unresolvable}, "permit") {
		t.Fatal("block list must take priority")
	}
}

func TestIPLiteralUsesPolicy(t *testing.T) {
	d := Decide("127.0.0.1", nil, nil, "permit", false)
	ips, ok := d.IPs, d.Allowed
	if !ok || len(ips) != 1 || ips[0].String() != "127.0.0.1" {
		t.Errorf("IP literal with permit default: ok=%v ips=%v", ok, ips)
	}
	if allowed("127.0.0.1", nil, nil, "block") {
		t.Error("IP literal with block default should be denied")
	}
	if allowed("127.0.0.1", nil, []string{"127.0.0.1"}, "permit") {
		t.Error("blocked IP literal should be denied")
	}
}

func TestBlockedIPDeniesEvenIfNamePermitted(t *testing.T) {
	if allowed("localhost", []string{"localhost"}, []string{"127.0.0.1"}, "permit") {
		t.Error("a resolved address on the block list must deny the request")
	}
}

func TestWildcardMatchesSubdomainsOnly(t *testing.T) {
	list := []string{"*.example.com"}
	cases := map[string]bool{
		"a.example.com":    true,
		"a.b.example.com":  true,
		"A.Example.COM.":   true,
		"example.com":      false,
		"evilexample.com":  false,
		"example.com.evil": false,
	}
	for host, want := range cases {
		if got := matchName(host, list); got != want {
			t.Errorf("matchName(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestExactNameDoesNotCoverSubdomains(t *testing.T) {
	if matchName("a.example.com", []string{"example.com"}) {
		t.Error("exact entry must not match subdomains; use *.example.com")
	}
}

func TestCIDRMatch(t *testing.T) {
	if !allowed("127.0.0.1", []string{"127.0.0.0/8"}, nil, "block") {
		t.Error("IP inside permitted CIDR should be allowed")
	}
	if allowed("127.0.0.1", nil, []string{"127.0.0.0/8"}, "permit") {
		t.Error("IP inside blocked CIDR should be denied")
	}
	if !allowed("127.0.0.1", nil, []string{"10.0.0.0/8"}, "permit") {
		t.Error("IP outside blocked CIDR should follow default")
	}
}

func TestResolvedAddressInBlockedCIDRDeniesName(t *testing.T) {
	if allowed("localhost", []string{"localhost"}, []string{"127.0.0.0/8"}, "permit") {
		t.Error("name resolving into a blocked CIDR must be denied even if permitted by name")
	}
}

func TestWildcardInPermitListAllowsSubdomainWithDefaultBlock(t *testing.T) {
	// localhost is not a subdomain of example.com, so it follows the default.
	if allowed("localhost", []string{"*.example.com"}, nil, "block") {
		t.Error("wildcard must not match unrelated names")
	}
}

func TestInternationalizedNamesMatchPunycode(t *testing.T) {
	cases := []struct {
		entry, host string
		want        bool
	}{
		{"münchen.de", "xn--mnchen-3ya.de", true},
		{"xn--mnchen-3ya.de", "MÜNCHEN.de.", true},
		{"*.münchen.de", "www.xn--mnchen-3ya.de", true},
		{"*.münchen.de", "münchen.de", false},
		{"münchen.de", "munchen.de", false},
	}
	for _, c := range cases {
		if got := matchName(c.host, []string{c.entry}); got != c.want {
			t.Errorf("matchName(%q, %q) = %v, want %v", c.host, c.entry, got, c.want)
		}
	}
}

func authorizeSpecial(host string, permit, block []string, def string) bool {
	d := Decide(host, permit, block, def, true)
	return d.Allowed
}

func TestSpecialRangesBlockedByDefaultPolicy(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "::1", "fe80::1", "fc00::1", "::ffff:10.0.0.1", "0.0.0.0"} {
		if authorizeSpecial(ip, nil, nil, "permit") {
			t.Errorf("special address %s allowed under default permit", ip)
		}
	}
}

func TestPublicAddressesNotAffectedBySpecialRanges(t *testing.T) {
	if !authorizeSpecial("8.8.8.8", nil, nil, "permit") {
		t.Error("public address should follow the policy")
	}
}

func TestExplicitIPPermitOverridesSpecialRange(t *testing.T) {
	if !authorizeSpecial("127.0.0.1", []string{"127.0.0.1"}, nil, "block") {
		t.Error("explicitly permitted loopback address should be allowed")
	}
	if !authorizeSpecial("10.0.0.0", []string{"10.0.0.0/8"}, nil, "block") {
		t.Error("explicitly permitted private CIDR should be allowed")
	}
}

func TestExplicitNamePermitAllowsResolvedSpecialAddress(t *testing.T) {
	if !authorizeSpecial("localhost", []string{"localhost"}, nil, "block") {
		t.Error("explicit name permit is honored even for special addresses")
	}
}

func TestSpecialRangesCanBeDisabled(t *testing.T) {
	if d := Decide("127.0.0.1", nil, nil, "permit", false); !d.Allowed {
		t.Error("with block_special_ranges off, loopback follows the default policy")
	}
}

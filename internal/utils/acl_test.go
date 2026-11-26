package utils

import "testing"

const unresolvable = "doppler-nonexistent-host.invalid"

func allowed(host string, permit, block []string, def string) bool {
	_, ok := Authorize(host, permit, block, def)
	return ok
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
	ips, ok := Authorize("127.0.0.1", nil, nil, "permit")
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

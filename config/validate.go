package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Validate returns one message per problem in c. An empty result means the
// configuration can be used. It checks syntax and ranges; it does not contact
// the network.
func Validate(c Config) []string {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	validListen := func(field, value string, required bool) {
		if value == "" {
			if required {
				add("%s: required", field)
			}
			return
		}
		_, port, err := net.SplitHostPort(value)
		if err != nil {
			add("%s: %q is not host:port", field, value)
			return
		}
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			add("%s: port %q is not in 1-65535", field, port)
		}
	}
	validListen("addr", c.Addr, true)
	validListen("tls_addr", c.TLSAddr, false)
	validListen("metrics_addr", c.MetricsAddr, false)

	if c.TLSAddr != "" {
		if c.TLSCert == "" || c.TLSKey == "" {
			add("tls_addr: tls_cert and tls_key are required")
		} else {
			for field, p := range map[string]string{"tls_cert": c.TLSCert, "tls_key": c.TLSKey} {
				if _, err := os.Stat(p); err != nil {
					add("%s: cannot read %q: %v", field, p, err)
				}
			}
		}
	}

	if d := strings.ToLower(c.ACL.Default); d != "permit" && d != "block" {
		add("ACL.default: must be \"permit\" or \"block\" (got %q)", c.ACL.Default)
	}
	for _, e := range c.ACL.Permit {
		if msg := validACLEntry(e); msg != "" {
			add("ACL.permit: %q %s", e, msg)
		}
	}
	for _, e := range c.ACL.Block {
		if msg := validACLEntry(e); msg != "" {
			add("ACL.block: %q %s", e, msg)
		}
	}

	for _, e := range c.MetricsAllow {
		if _, _, err := net.ParseCIDR(e); err != nil && net.ParseIP(e) == nil {
			add("metrics_allow: %q is not an IP or CIDR range", e)
		}
	}
	if c.PublicHost != "" && (strings.Contains(c.PublicHost, "/") || strings.ContainsAny(c.PublicHost, " \t")) {
		add("public_host: %q must be a host name or IP only", c.PublicHost)
	}

	nonNeg := map[string]int64{
		"limit":                       int64(c.Limit),
		"read_timeout":                int64(c.ReadTimeout),
		"write_timeout":               int64(c.WriteTimeout),
		"idle_timeout":                int64(c.IdleTimeout),
		"max_connections":             int64(c.MaxConnections),
		"max_tunnel_seconds":          int64(c.MaxTunnelSeconds),
		"max_header_bytes":            int64(c.MaxHeaderBytes),
		"dial_timeout":                int64(c.DialTimeout),
		"auth.max_failures":           int64(c.Auth.MaxFailures),
		"auth.failure_window_minutes": int64(c.Auth.FailureWindowMin),
		"auth.block_minutes":          int64(c.Auth.BlockMin),
		"auth.password.min_length":    int64(c.Auth.Password.MinLength),
	}
	for field, v := range nonNeg {
		if v < 0 {
			add("%s: must not be negative (got %d)", field, v)
		}
	}

	if c.Auth.Enabled {
		if c.TLSAddr == "" {
			add("auth.enabled: requires tls_addr; credentials are only accepted over TLS")
		}
		if len(c.Auth.Users) == 0 {
			add("auth.enabled: no users defined")
		}
		for user, hash := range c.Auth.Users {
			if user == "" || strings.Contains(user, ":") {
				add("auth.users: invalid user name %q", user)
			}
			if !strings.HasPrefix(hash, "$2") {
				add("auth.users.%s: not a bcrypt hash (generate one with cmd/hashpw)", user)
				continue
			}
			if _, err := bcrypt.Cost([]byte(hash)); err != nil {
				add("auth.users.%s: malformed bcrypt hash: %v", user, err)
			}
		}
	}
	return errs
}

// validACLEntry returns "" for a valid permit/block entry, or the reason it is not.
func validACLEntry(e string) string {
	if strings.TrimSpace(e) != e || e == "" {
		return "is empty or has surrounding spaces"
	}
	if strings.Contains(e, "/") {
		if _, _, err := net.ParseCIDR(e); err != nil {
			return "is not a valid CIDR range"
		}
		return ""
	}
	if net.ParseIP(e) != nil {
		return ""
	}
	name := e
	if strings.HasPrefix(name, "*.") {
		name = name[2:]
	}
	if name == "" || strings.Contains(name, "*") || strings.ContainsAny(name, " \t") {
		return "is not a valid host name (wildcards are only allowed as a leading \"*.\")"
	}
	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		if label == "" {
			return "has an empty label"
		}
	}
	return ""
}

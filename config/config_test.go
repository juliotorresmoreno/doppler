package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hasError(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

func TestValidateDefaultsAreValid(t *testing.T) {
	if errs := Validate(defaults()); len(errs) != 0 {
		t.Fatalf("defaults rejected: %v", errs)
	}
}

func TestValidateReportsEachProblem(t *testing.T) {
	cases := []struct {
		name string
		edit func(c *Config)
		want string
	}{
		{"bad addr", func(c *Config) { c.Addr = "5000" }, "addr"},
		{"bad default", func(c *Config) { c.ACL.Default = "allow" }, "ACL.default"},
		{"bad CIDR", func(c *Config) { c.ACL.Permit = []string{"10.0.0.0/33"} }, "ACL.permit"},
		{"misplaced wildcard", func(c *Config) { c.ACL.Block = []string{"a*.example.com"} }, "ACL.block"},
		{"negative limit", func(c *Config) { c.MaxConnections = -1 }, "max_connections"},
		{"metrics allow", func(c *Config) { c.MetricsAllow = []string{"not-an-ip"} }, "metrics_allow"},
		{"auth without tls", func(c *Config) {
			c.Auth.Enabled = true
			c.Auth.Users = map[string]string{"julio": "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ0123"}
		}, "requires tls_addr"},
		{"plaintext password", func(c *Config) {
			c.TLSAddr, c.TLSCert, c.TLSKey = ":5443", "/nonexistent.crt", "/nonexistent.key"
			c.Auth.Enabled = true
			c.Auth.Users = map[string]string{"julio": "123456"}
		}, "not a bcrypt hash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := defaults()
			tc.edit(&c)
			if errs := Validate(c); !hasError(errs, tc.want) {
				t.Errorf("expected an error mentioning %q, got %v", tc.want, errs)
			}
		})
	}
}

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReloadAppliesPolicyAndKeepsRestartOnlySettings(t *testing.T) {
	dir := t.TempDir()
	configPath = writeConfig(t, dir, "addr: \":5000\"\nACL:\n  default: block\n")
	initial, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	Set(initial)
	if c, _ := GetConfig(); c.ACL.Default != "block" {
		t.Fatalf("policy not loaded: %q", c.ACL.Default)
	}

	configPath = writeConfig(t, dir, "addr: \":6000\"\nACL:\n  default: permit\n")
	warnings, err := Reload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	c, _ := GetConfig()
	if c.ACL.Default != "permit" {
		t.Errorf("policy change not applied: %q", c.ACL.Default)
	}
	if c.Addr != ":5000" {
		t.Errorf("restart-only addr changed at runtime: %q", c.Addr)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "addr") {
		t.Errorf("expected a restart warning for addr, got %v", warnings)
	}
}

func TestReloadRejectsInvalidFileAndKeepsCurrent(t *testing.T) {
	dir := t.TempDir()
	configPath = writeConfig(t, dir, "ACL:\n  default: block\n")
	initial, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	Set(initial)

	configPath = writeConfig(t, dir, "ACL:\n  default: allow\n")
	if _, err := Reload(); err == nil || !strings.Contains(err.Error(), "ACL.default") {
		t.Fatalf("invalid file should be rejected, got %v", err)
	}
	if c, _ := GetConfig(); c.ACL.Default != "block" {
		t.Errorf("rejected reload changed the running policy to %q", c.ACL.Default)
	}
}

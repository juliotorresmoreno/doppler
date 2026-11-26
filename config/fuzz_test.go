package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

// FuzzACLEntry: validation of a single permit/block entry must not panic.
func FuzzACLEntry(f *testing.F) {
	for _, s := range []string{"example.com", "*.example.com", "10.0.0.0/8", "::1", "*", "a..b", "", " x "} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, entry string) {
		_ = validACLEntry(entry)
	})
}

// FuzzConfigYAML: any YAML document must parse or fail cleanly, and validation of
// whatever parses must not panic.
func FuzzConfigYAML(f *testing.F) {
	f.Add([]byte("addr: \":5000\"\nACL:\n  default: block\n  permit:\n    - \"*.example.com\"\n"))
	f.Add([]byte("auth:\n  enabled: true\n  users:\n    julio: \"$2a$10$abc\"\n"))
	f.Add([]byte("{{{"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, doc []byte) {
		var c Config
		if err := yaml.Unmarshal(doc, &c); err != nil {
			return
		}
		_ = Validate(c)
	})
}

package config

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Credentials struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Auth struct {
	Enabled          bool              `yaml:"enabled"`
	Users            map[string]string `yaml:"users"`
	MaxFailures      int               `yaml:"max_failures"`
	FailureWindowMin int               `yaml:"failure_window_minutes"`
	BlockMin         int               `yaml:"block_minutes"`
	Password         PasswordPolicy    `yaml:"password"`
}

// PasswordPolicy is enforced by cmd/hashpw when a new credential is created.
type PasswordPolicy struct {
	MinLength     int  `yaml:"min_length"`
	RequireUpper  bool `yaml:"require_upper"`
	RequireDigit  bool `yaml:"require_digit"`
	RequireSymbol bool `yaml:"require_symbol"`
}

type ACL struct {
	Default string   `yaml:"default"`
	Permit  []string `yaml:"permit"`
	Block   []string `yaml:"block"`
	// BlockSpecialRanges denies loopback, private, link-local and other reserved
	// addresses unless they are explicitly permitted by IP or CIDR.
	BlockSpecialRanges bool `yaml:"block_special_ranges"`
}

type Config struct {
	Auth         Auth   `yaml:"auth"`
	ACL          ACL    `yaml:"ACL"`
	Limit        int64  `yaml:"limit"`
	Addr         string `yaml:"addr"`
	TLSAddr      string `yaml:"tls_addr"`
	TLSCert      string `yaml:"tls_cert"`
	TLSKey       string `yaml:"tls_key"`
	PublicHost   string `yaml:"public_host"` // host clients use to reach this proxy, e.g. proxy.example.com
	Logger       bool   `yaml:"logger"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
	IdleTimeout  int    `yaml:"idle_timeout"` // seconds without traffic before a tunnel is closed; 0 disables
	// MaxConnections caps concurrent proxied connections (tunnels and forwarded
	// requests). Extra requests get 503. 0 means no limit.
	MaxConnections int `yaml:"max_connections"`
	// MaxTunnelSeconds closes a tunnel after this many seconds, whatever its
	// activity. 0 means no limit.
	MaxTunnelSeconds int `yaml:"max_tunnel_seconds"`
	// MaxHeaderBytes bounds the size of request headers. 0 keeps the Go default (1 MB).
	MaxHeaderBytes int `yaml:"max_header_bytes"`
}

var config interface{}
var configPath string = ""

func getConfigArgs() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	configPathDefault := path.Join(dir, "config.yml")
	flag.StringVar(&configPath, "c", configPathDefault, "config path")
	flag.Parse()
}

// PACFile returns the proxy.pac path, expected next to the config file.
func PACFile() string {
	if configPath == "" {
		getConfigArgs()
	}
	dir, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return "proxy.pac"
	}
	return filepath.Join(dir, "proxy.pac")
}

func GetConfig() (Config, error) {
	if config != nil {
		return config.(Config), nil
	}

	if configPath == "" {
		getConfigArgs()
	}
	result := Config{
		Addr:           ":4080",
		Limit:          10,
		Logger:         true,
		ReadTimeout:    30,
		WriteTimeout:   120,
		IdleTimeout:    300,
		MaxConnections: 1024,
		MaxHeaderBytes: 16 << 10,
		Auth: Auth{
			Enabled:          false,
			Users:            make(map[string]string),
			MaxFailures:      5,
			FailureWindowMin: 15,
			BlockMin:         15,
			Password: PasswordPolicy{
				MinLength:     10,
				RequireUpper:  true,
				RequireDigit:  true,
				RequireSymbol: true,
			},
		},
		ACL: ACL{
			Default:            "permit",
			Permit:             make([]string, 0),
			Block:              make([]string, 0),
			BlockSpecialRanges: true,
		},
	}

	f, err := os.Open(configPath)
	if err != nil {
		return result, err
	}
	buff, err := io.ReadAll(f)
	if err != nil {
		return result, err
	}
	err = yaml.Unmarshal(buff, &result)
	if err != nil {
		return result, err
	}
	config = result

	return config.(Config), nil
}

// Set replaces the loaded configuration. It is meant for tests and for code
// that embeds the proxy handler.
func Set(c Config) {
	config = c
}

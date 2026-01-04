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
	Enabled bool              `yaml:"enabled"`
	Users   map[string]string `yaml:"users"`
}

type ACL struct {
	Default string   `yaml:"default"`
	Permit  []string `yaml:"permit"`
	Block   []string `yaml:"block"`
}

type Config struct {
	Auth         Auth   `yaml:"auth"`
	ACL          ACL    `yaml:"ACL"`
	Limit        int64  `yaml:"limit"`
	Addr         string `yaml:"addr"`
	Logger       bool   `yaml:"logger"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
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

func GetConfig() (Config, error) {
	if config != nil {
		return config.(Config), nil
	}

	if configPath == "" {
		getConfigArgs()
	}
	result := Config{
		Addr:         ":4080",
		Limit:        10,
		Logger:       true,
		ReadTimeout:  30,
		WriteTimeout: 120,
		Auth: Auth{
			Enabled: false,
			Users:   make(map[string]string),
		},
		ACL: ACL{
			Default: "permit",
			Permit:  make([]string, 0),
			Block:   make([]string, 0),
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

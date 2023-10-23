package config

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Limit        int               `yaml:"limit"`
	Addr         string            `yaml:"addr"`
	Logger       bool              `yaml:"logger"`
	Database     map[string]string `yaml:"database"`
	ReadTimeout  int               `yaml:"read_timeout"`
	WriteTimeout int               `yaml:"write_timeout"`
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
	}

	f, err := os.Open(configPath)
	if err != nil {
		return result, err
	}
	buff, err := ioutil.ReadAll(f)
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

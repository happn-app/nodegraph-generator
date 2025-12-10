package pkg

import (
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Host string `yaml:"host"`
	Timeout time.Duration `yaml:"timeout"`
	PrometheusHost string `yaml:"prometheus_host"`
	queryRange time.Duration `yaml:"query_range"`
	queryStep time.Duration `yaml:"query_step"`
}

func LoadConfig() Config {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "/etc/nodegraph-generator/config.yaml"
	}
	configBytes, err := os.ReadFile(path);
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}
	return config
}

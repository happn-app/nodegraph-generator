package pkg

import (
	"log"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Host           string        `yaml:"host"`
	Timeout        time.Duration `yaml:"timeout"`
	PrometheusHost string        `yaml:"prometheus_host"`
	QueryRange     time.Duration `yaml:"query_range"`
	QueryStep      time.Duration `yaml:"query_step"`
}

func LoadConfig() Config {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "/etc/nodegraph-generator/config.yaml"
	}
	configBytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}
	log.Printf("[LoadConfig] Loaded config: %+v", config)
	log.Printf("[LoadConfig] Config path: %s", path)
	return config
}

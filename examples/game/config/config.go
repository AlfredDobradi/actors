package config

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

var cfg *Config

func GetConfig() *Config {
	if cfg == nil {
		slog.Warn("Config not loaded, returning default values")
		return &Config{
			Addr:     "localhost:8080",
			NodeName: "default-node",
		}
	}
	return cfg
}

type Config struct {
	Addr     string `yaml:"node_address"`
	NodeName string `yaml:"node_name"`
}

func Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	return nil
}

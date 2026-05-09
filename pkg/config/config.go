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
		cfg = &Config{
			Addr:     "localhost:8080",
			NodeName: "default-node",
		}
	}
	return cfg
}

type Logging struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AddSource bool   `yaml:"add_source"`
}

type Config struct {
	Addr     string  `yaml:"node_address"`
	NodeName string  `yaml:"node_name"`
	Logging  Logging `yaml:"logging"`
}

func Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var tmpCfg Config
	if err := yaml.Unmarshal(raw, &tmpCfg); err != nil {
		return err
	}
	cfg = &tmpCfg

	slog.Info("Config loaded successfully", "config", cfg)
	return nil
}

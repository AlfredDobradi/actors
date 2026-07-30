package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

type Database struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"dbname"`
}

func (db Database) DSN() string {
	pairs := make([]string, 0, 6)

	if db.Host != "" {
		pairs = append(pairs, fmt.Sprintf("host=%s", db.Host))
	}

	if db.User != "" {
		pairs = append(pairs, fmt.Sprintf("user=%s", db.User))
	}

	if db.Password != "" {
		pairs = append(pairs, fmt.Sprintf("password=%s", db.Password))
	}

	if db.Port != "" {
		pairs = append(pairs, fmt.Sprintf("port=%s", db.Port))
	}

	if db.DBName != "" {
		pairs = append(pairs, fmt.Sprintf("dbname=%s", db.DBName))
	}

	pairs = append(pairs, "sslmode=disable")

	return strings.Join(pairs, " ")
}

type KV struct {
	Hosts []string `yaml:"hosts"`
}

type Config struct {
	Addr     string   `yaml:"node_address"`
	NodeName string   `yaml:"node_name"`
	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`
	KV       KV       `yaml:"kv"`
}

func Load(path string) error {
	path = filepath.Clean(path)
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

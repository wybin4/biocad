package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Watcher  WatcherConfig  `yaml:"watcher"`
	API      APIConfig      `yaml:"api"`
}

type DatabaseConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	AuthDB   string `yaml:"auth_db"`
}

type WatcherConfig struct {
	InputDir     string        `yaml:"input_dir"`
	OutputDir    string        `yaml:"output_dir"`
	PollInterval time.Duration `yaml:"poll_interval"`
	Workers      int           `yaml:"workers"`
}

type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func (c *DatabaseConfig) GetMongoURI() string {
	if c.URI != "" {
		return c.URI
	}
	return "mongodb://localhost:27017"
}

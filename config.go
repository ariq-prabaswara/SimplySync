package main

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Paths  PathsConfig  `toml:"paths"`
	Ignore IgnoreConfig `toml:"ignore"`
}

type PathsConfig struct {
	Source      string `toml:"source"`
	Destination string `toml:"destination"`
}

type IgnoreConfig struct {
	Patterns []string `toml:"patterns"`
}

func LoadConfig(exeDir string) (*Config, error) {
	path := filepath.Join(exeDir, "sync.toml")
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("cannot load sync.toml: %w", err)
	}
	if cfg.Paths.Source == "" || cfg.Paths.Destination == "" {
		return nil, fmt.Errorf("sync.toml: [paths] source and destination must not be empty")
	}
	return &cfg, nil
}

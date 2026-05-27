package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Fresh   FreshConfig   `yaml:"fresh"`
	Secrets SecretsConfig `yaml:"secrets"`
	Scan    ScanConfig    `yaml:"scan"`
}

type FreshConfig struct {
	AgeGateDays *int   `yaml:"age_gate_days"`
	IncludeDev  *bool  `yaml:"include_dev"`
	FailOn      string `yaml:"fail_on"`
	Format      string `yaml:"format"`
}

type SecretsConfig struct {
	Tool string `yaml:"tool"`
}

type ScanConfig struct {
	Tools []string `yaml:"tools"`
}

func Load(path string) (Config, error) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	configPath := filepath.Join(dir, ".zick.yaml")
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("stat %s: %w", configPath, err)
	}

	return parse(configPath)
}

func parse(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

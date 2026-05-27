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
	SBOM    SBOMConfig    `yaml:"sbom"`
	Hook    HookConfig    `yaml:"hook"`
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
	Tools       []string `yaml:"tools"`
	SARIFOutput string   `yaml:"sarif_output"`
}

type SBOMConfig struct {
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type HookConfig struct {
	IncludeSecrets *bool  `yaml:"include_secrets"`
	SecretsTool    string `yaml:"secrets_tool"`
}

func Load(path string) (Config, error) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	configPath, err := findConfig(dir)
	if err != nil {
		return Config{}, err
	}
	if configPath == "" {
		return Config{}, nil
	}

	return parse(configPath)
}

func findConfig(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve config search path %s: %w", dir, err)
	}

	for {
		configPath := filepath.Join(abs, ".zick.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", configPath, err)
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", nil
		}
		abs = parent
	}
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

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
	Report  ReportConfig  `yaml:"report"`
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

type ReportConfig struct {
	JSONOutput string `yaml:"json_output"`
	HTMLOutput string `yaml:"html_output"`
}

// Load returns the effective config for the given path.
//
// Resolution order (highest to lowest priority):
//  1. Per-repo .zick.yaml — found by walking up from path
//  2. Global config — ~/.config/zick/config.yaml (os.UserConfigDir)
//  3. Command flag defaults
func Load(path string) (Config, error) {
	global, err := loadGlobal()
	if err != nil {
		return Config{}, err
	}

	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	configPath, err := findConfig(dir)
	if err != nil {
		return Config{}, err
	}
	if configPath == "" {
		return global, nil
	}

	repo, err := parse(configPath)
	if err != nil {
		return Config{}, err
	}
	return merge(global, repo), nil
}

func loadGlobal() (Config, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, nil
	}
	path := filepath.Join(cfgDir, "zick", "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}
	return parse(path)
}

// merge returns base with any explicitly-set fields in override applied on top.
func merge(base, override Config) Config {
	if override.Fresh.AgeGateDays != nil {
		base.Fresh.AgeGateDays = override.Fresh.AgeGateDays
	}
	if override.Fresh.IncludeDev != nil {
		base.Fresh.IncludeDev = override.Fresh.IncludeDev
	}
	if override.Fresh.FailOn != "" {
		base.Fresh.FailOn = override.Fresh.FailOn
	}
	if override.Fresh.Format != "" {
		base.Fresh.Format = override.Fresh.Format
	}
	if override.Secrets.Tool != "" {
		base.Secrets.Tool = override.Secrets.Tool
	}
	if len(override.Scan.Tools) > 0 {
		base.Scan.Tools = override.Scan.Tools
	}
	if override.Scan.SARIFOutput != "" {
		base.Scan.SARIFOutput = override.Scan.SARIFOutput
	}
	if override.SBOM.Format != "" {
		base.SBOM.Format = override.SBOM.Format
	}
	if override.SBOM.Output != "" {
		base.SBOM.Output = override.SBOM.Output
	}
	if override.Hook.IncludeSecrets != nil {
		base.Hook.IncludeSecrets = override.Hook.IncludeSecrets
	}
	if override.Hook.SecretsTool != "" {
		base.Hook.SecretsTool = override.Hook.SecretsTool
	}
	if override.Report.JSONOutput != "" {
		base.Report.JSONOutput = override.Report.JSONOutput
	}
	if override.Report.HTMLOutput != "" {
		base.Report.HTMLOutput = override.Report.HTMLOutput
	}
	return base
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

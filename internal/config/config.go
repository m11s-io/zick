package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
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
//  2. Global config — os.UserConfigDir()/zick/config.yaml
//  3. Command flag defaults
func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Load global config as base layer.
	if globalPath, err := globalConfigPath(); err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			v.SetConfigFile(globalPath)
			if err := v.ReadInConfig(); err != nil {
				return Config{}, fmt.Errorf("parse global config: %w", err)
			}
		}
	}

	// Find per-repo config and merge on top.
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}
	repoPath, err := findConfig(dir)
	if err != nil {
		return Config{}, err
	}
	if repoPath != "" {
		v.SetConfigFile(repoPath)
		if err := v.MergeInConfig(); err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", repoPath, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, useYAMLTags); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// useYAMLTags tells mapstructure to use yaml struct tags for field name
// mapping so we don't need duplicate mapstructure tags on every field.
func useYAMLTags(c *mapstructure.DecoderConfig) {
	c.TagName = "yaml"
}

func globalConfigPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "zick", "config.yaml"), nil
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

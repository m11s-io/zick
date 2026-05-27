package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	config := `# zick config
fresh:
  age_gate_days: 14
  include_dev: true
  fail_on: warn

secrets:
  tool: gitleaks

scan:
  tools: [osv-scanner, trivy]

hook:
  include_secrets: true
  secrets_tool: gitleaks

report:
  json_output: zick-report.json
  html_output: zick-report.html
`
	if err := os.WriteFile(filepath.Join(dir, ".zick.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Fresh.AgeGateDays == nil || *cfg.Fresh.AgeGateDays != 14 {
		t.Fatalf("age_gate_days = %v, want 14", cfg.Fresh.AgeGateDays)
	}
	if cfg.Fresh.IncludeDev == nil || !*cfg.Fresh.IncludeDev {
		t.Fatalf("include_dev = %v, want true", cfg.Fresh.IncludeDev)
	}
	if cfg.Fresh.FailOn != "warn" {
		t.Fatalf("fail_on = %q, want warn", cfg.Fresh.FailOn)
	}
	if cfg.Secrets.Tool != "gitleaks" {
		t.Fatalf("secrets.tool = %q, want gitleaks", cfg.Secrets.Tool)
	}
	if len(cfg.Scan.Tools) != 2 || cfg.Scan.Tools[0] != "osv-scanner" || cfg.Scan.Tools[1] != "trivy" {
		t.Fatalf("scan.tools = %v, want [osv-scanner trivy]", cfg.Scan.Tools)
	}
	if cfg.Hook.IncludeSecrets == nil || !*cfg.Hook.IncludeSecrets {
		t.Fatalf("hook.include_secrets = %v, want true", cfg.Hook.IncludeSecrets)
	}
	if cfg.Hook.SecretsTool != "gitleaks" {
		t.Fatalf("hook.secrets_tool = %q, want gitleaks", cfg.Hook.SecretsTool)
	}
	if cfg.Report.JSONOutput != "zick-report.json" {
		t.Fatalf("report.json_output = %q, want zick-report.json", cfg.Report.JSONOutput)
	}
	if cfg.Report.HTMLOutput != "zick-report.html" {
		t.Fatalf("report.html_output = %q, want zick-report.html", cfg.Report.HTMLOutput)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Fresh.AgeGateDays != nil || cfg.Fresh.IncludeDev != nil || cfg.Fresh.FailOn != "" || cfg.Secrets.Tool != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestLoadFromFilePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zick.yaml"), []byte("secrets:\n  tool: betterleaks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "package.json")
	if err := os.WriteFile(manifest, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(manifest)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Secrets.Tool != "betterleaks" {
		t.Fatalf("secrets.tool = %q, want betterleaks", cfg.Secrets.Tool)
	}
}

func TestGlobalAndRepoMerge(t *testing.T) {
	// Set up a temp global config dir.
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	globalDir := filepath.Join(cfgDir, "zick")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(globalDir, "config.yaml"),
		"fresh:\n  age_gate_days: 14\n  fail_on: warn\nsecrets:\n  tool: gitleaks\n")

	// Per-repo config overrides only age_gate_days; other fields inherit from global.
	repoDir := t.TempDir()
	writeFile(t, filepath.Join(repoDir, ".zick.yaml"), "fresh:\n  age_gate_days: 3\n")

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Fresh.AgeGateDays == nil || *cfg.Fresh.AgeGateDays != 3 {
		t.Fatalf("age_gate_days = %v, want 3 (repo wins)", cfg.Fresh.AgeGateDays)
	}
	if cfg.Fresh.FailOn != "warn" {
		t.Fatalf("fail_on = %q, want warn (global survives)", cfg.Fresh.FailOn)
	}
	if cfg.Secrets.Tool != "gitleaks" {
		t.Fatalf("secrets.tool = %q, want gitleaks (global survives)", cfg.Secrets.Tool)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadWalksUpFromNestedPath(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".zick.yaml"), []byte("sbom:\n  format: spdx-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nested)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SBOM.Format != "spdx-json" {
		t.Fatalf("sbom.format = %q, want spdx-json", cfg.SBOM.Format)
	}
}

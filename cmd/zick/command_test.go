package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m11s-io/zick/internal/fresh"
	"github.com/spf13/cobra"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestFreshRejectsInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"age gate", []string{"fresh", "--age-gate", "0"}, "--age-gate must be greater than 0"},
		{"fail on", []string{"fresh", "--fail-on", "low"}, "--fail-on must be one of"},
		{"format", []string{"fresh", "--format", "xml"}, "--format must be one of"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := executeForTest(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestFreshLoadsConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".zick.yaml"), "fresh:\n  age_gate_days: 0\n")

	_, _, err := executeForTest(t, "fresh", dir)
	if err == nil || !strings.Contains(err.Error(), "--age-gate must be greater than 0") {
		t.Fatalf("error = %v, want config age gate validation", err)
	}
}

func TestFreshFlagsOverrideConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".zick.yaml"), "fresh:\n  age_gate_days: 0\n")

	out, _, err := executeForTest(t, "fresh", "--age-gate", "7", dir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "No supported manifest found") {
		t.Fatalf("stdout = %q, want no manifest message", out)
	}
}

func TestSecretsRejectsInvalidTool(t *testing.T) {
	_, _, err := executeForTest(t, "secrets", "--tool", "detect-secrets")
	if err == nil || !strings.Contains(err.Error(), "--tool must be one of") {
		t.Fatalf("error = %v, want invalid tool error", err)
	}
}

func TestScanRejectsInvalidTool(t *testing.T) {
	_, _, err := executeForTest(t, "scan", "--tools", "bad")
	if err == nil || !strings.Contains(err.Error(), "unsupported scanner") {
		t.Fatalf("error = %v, want unsupported scanner error", err)
	}
}

func TestSBOMRejectsInvalidFormat(t *testing.T) {
	_, _, err := executeForTest(t, "sbom", "--format", "xml")
	if err == nil || !strings.Contains(err.Error(), "--format must be one of") {
		t.Fatalf("error = %v, want invalid format error", err)
	}
}

func TestAuditRejectsInvalidScanTool(t *testing.T) {
	_, _, err := executeForTest(t, "audit", "--scan-tools", "bad")
	if err == nil || !strings.Contains(err.Error(), "unsupported scanner") {
		t.Fatalf("error = %v, want unsupported scanner error", err)
	}
}

func TestHookInstallUsesConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, ".zick.yaml"), "hook:\n  include_secrets: true\n  secrets_tool: gitleaks\n")

	out, _, err := executeForTest(t, "hook", "install", dir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Installed pre-commit hook") {
		t.Fatalf("stdout = %q, want install message", out)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), `zick secrets --tool "gitleaks" .`) {
		t.Fatalf("hook script = %q, want configured secrets tool", string(data))
	}
}

func TestHookInstallRejectsInvalidSecretsTool(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := executeForTest(t, "hook", "install", "--secrets-tool", "detect-secrets", dir)
	if err == nil || !strings.Contains(err.Error(), "--tool must be one of") {
		t.Fatalf("error = %v, want invalid secrets tool error", err)
	}
}

func TestSplitTools(t *testing.T) {
	got := splitTools("osv-scanner, trivy,,")
	if len(got) != 2 || got[0] != "osv-scanner" || got[1] != "trivy" {
		t.Fatalf("splitTools = %v, want [osv-scanner trivy]", got)
	}
}

func TestPrintFreshJSON(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := printFreshResults(cmd, []fresh.Result{{
		Package:   "lodash",
		Version:   "4.17.21",
		Published: time.Date(2021, 2, 20, 0, 0, 0, 0, time.UTC),
		Age:       24 * time.Hour,
		Risk:      fresh.RiskOK,
	}}, "json")
	if err != nil {
		t.Fatalf("printFreshResults: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"risk": "OK"`) || !strings.Contains(got, `"package": "lodash"`) {
		t.Fatalf("json output = %q", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunSecretsUsesLocalGitleaks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "gitleaks")
	writeExecutable(t, bin, "#!/bin/sh\necho \"$0 $@\"\n")
	t.Setenv("PATH", dir)

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunSecrets("/repo", "gitleaks"); err != nil {
		t.Fatalf("RunSecrets: %v", err)
	}
	if !strings.Contains(out.String(), "detect --source /repo --no-banner") {
		t.Fatalf("stdout = %q, want gitleaks args", out.String())
	}
}

func TestRunScanRunsToolsInOrder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "osv-scanner"), "#!/bin/sh\necho osv \"$@\"\n")
	writeExecutable(t, filepath.Join(dir, "trivy"), "#!/bin/sh\necho trivy \"$@\"\n")
	t.Setenv("PATH", dir)

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunScan("/repo", []string{"osv-scanner", "trivy"}, ScanOptions{}); err != nil {
		t.Fatalf("RunScan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Running osv-scanner") || !strings.Contains(got, "osv --recursive /repo") {
		t.Fatalf("stdout = %q, want osv scanner execution", got)
	}
	if !strings.Contains(got, "Running trivy") || !strings.Contains(got, "trivy fs /repo") {
		t.Fatalf("stdout = %q, want trivy execution", got)
	}
}

func TestRunScanPassesSARIFOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "trivy"), "#!/bin/sh\necho trivy \"$@\"\n")
	t.Setenv("PATH", dir)

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunScan("/repo", []string{"trivy"}, ScanOptions{SARIFOutput: "scan.sarif"}); err != nil {
		t.Fatalf("RunScan: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "trivy fs /repo --format sarif --output scan.sarif") {
		t.Fatalf("stdout = %q, want SARIF args", got)
	}
}

func TestRunSBOMUsesSyftArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "syft"), "#!/bin/sh\necho syft \"$@\"\n")
	t.Setenv("PATH", dir)

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunSBOM("/repo", SBOMOptions{Format: "spdx-json", Output: "sbom.json"}); err != nil {
		t.Fatalf("RunSBOM: %v", err)
	}

	if !strings.Contains(out.String(), "syft /repo -o spdx-json=sbom.json") {
		t.Fatalf("stdout = %q, want syft args", out.String())
	}
}

func TestRunFallsBackToDockerWithAbsoluteMount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	docker := filepath.Join(dir, "docker")
	writeExecutable(t, docker, "#!/bin/sh\necho docker \"$@\"\n")
	t.Setenv("PATH", dir)

	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunSecrets(repo, "gitleaks"); err != nil {
		t.Fatalf("RunSecrets: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "-v "+repo+":/src") {
		t.Fatalf("stdout = %q, want absolute Docker mount", got)
	}
	if !strings.Contains(got, "detect --source /src --no-banner") {
		t.Fatalf("stdout = %q, want container path args", got)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

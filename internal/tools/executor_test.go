package tools

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/m11s-io/zick/internal/cli"
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
	if !strings.Contains(got, "-w /src") {
		t.Fatalf("stdout = %q, want container workdir", got)
	}
	if !strings.Contains(got, "GIT_CONFIG_KEY_0=safe.directory") || !strings.Contains(got, "GIT_CONFIG_VALUE_0=/src") {
		t.Fatalf("stdout = %q, want Git safe.directory Docker env", got)
	}
	if !strings.Contains(got, "detect --source . --no-banner") {
		t.Fatalf("stdout = %q, want container path args", got)
	}
}

func TestRunSecretsAutoFallsBackToBetterleaksDocker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	docker := filepath.Join(dir, "docker")
	writeExecutable(t, docker, "#!/bin/sh\necho docker \"$@\"\n")
	t.Setenv("PATH", dir)

	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunSecrets(repo, "auto"); err != nil {
		t.Fatalf("RunSecrets: %v", err)
	}

	got := out.String()
	if strings.Contains(got, "ghcr.io/smartbugs/betterleaks") {
		t.Fatalf("stdout = %q, should not use deprecated betterleaks Docker image", got)
	}
	if !strings.Contains(got, "ghcr.io/betterleaks/betterleaks:latest") {
		t.Fatalf("stdout = %q, want betterleaks Docker fallback", got)
	}
	if !strings.Contains(got, "git .") {
		t.Fatalf("stdout = %q, want betterleaks git subcommand with container path", got)
	}
}

func TestRunSecretsBetterleaksUsesDirForNonGitTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "betterleaks"), "#!/bin/sh\necho betterleaks \"$@\"\n")
	t.Setenv("PATH", dir)

	target := filepath.Join(t.TempDir(), "plain")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	if err := executor.RunSecrets(target, "betterleaks"); err != nil {
		t.Fatalf("RunSecrets: %v", err)
	}

	if !strings.Contains(out.String(), "dir "+target) {
		t.Fatalf("stdout = %q, want betterleaks dir subcommand", out.String())
	}
}

func TestRunReturnsSilentErrorForToolExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper shell script is POSIX-only")
	}

	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "gitleaks"), "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", dir)

	var out, errOut bytes.Buffer
	executor := NewExecutor(&out, &errOut)
	err := executor.RunSecrets("/repo", "gitleaks")

	var silent *cli.SilentError
	if !errors.As(err, &silent) || silent.Code != 1 {
		t.Fatalf("error = %v, want SilentError code 1", err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

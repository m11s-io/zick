package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesPreCommitHook(t *testing.T) {
	dir := gitRepo(t)

	hookPath, err := Install(dir, InstallOptions{IncludeSecrets: true, SecretsTool: "gitleaks"})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "zick fresh .") || !strings.Contains(got, `zick secrets --tool "gitleaks" .`) {
		t.Fatalf("hook script = %q, want fresh and secrets commands", got)
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestInstallRefusesUnmanagedHook(t *testing.T) {
	dir := gitRepo(t)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nmake test\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Install(dir, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "not managed by zick") {
		t.Fatalf("Install error = %v, want unmanaged hook error", err)
	}
}

func TestInstallForceReplacesUnmanagedHook(t *testing.T) {
	dir := gitRepo(t)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nmake test\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(dir, InstallOptions{Force: true}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), beginMarker) {
		t.Fatalf("hook script = %q, want zick marker", string(data))
	}
}

func TestUninstallRemovesManagedHook(t *testing.T) {
	dir := gitRepo(t)
	hookPath, err := Install(dir, InstallOptions{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if _, err := Uninstall(dir, false); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("Stat error = %v, want hook removed", err)
	}
}

func TestUninstallKeepsUnmanagedHook(t *testing.T) {
	dir := gitRepo(t)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nmake test\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Uninstall(dir, false)
	if err == nil || !strings.Contains(err.Error(), "not managed by zick") {
		t.Fatalf("Uninstall error = %v, want unmanaged hook error", err)
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

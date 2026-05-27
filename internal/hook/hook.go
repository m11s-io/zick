package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	beginMarker = "# zick hook begin"
	endMarker   = "# zick hook end"
)

type InstallOptions struct {
	IncludeSecrets bool
	SecretsTool    string
	Force          bool
}

func Install(path string, opts InstallOptions) (string, error) {
	hookPath, err := preCommitPath(path)
	if err != nil {
		return "", err
	}

	if data, err := os.ReadFile(hookPath); err == nil {
		if !isManaged(string(data)) && !opts.Force {
			return "", fmt.Errorf("%s already exists and is not managed by zick; rerun with --force to replace it", hookPath)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", hookPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		return "", fmt.Errorf("create hook directory: %w", err)
	}

	if opts.SecretsTool == "" {
		opts.SecretsTool = "auto"
	}
	if err := os.WriteFile(hookPath, []byte(script(opts)), 0o755); err != nil {
		return "", fmt.Errorf("write %s: %w", hookPath, err)
	}
	return hookPath, nil
}

func Uninstall(path string, force bool) (string, error) {
	hookPath, err := preCommitPath(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(hookPath)
	if os.IsNotExist(err) {
		return hookPath, nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", hookPath, err)
	}
	if !isManaged(string(data)) && !force {
		return "", fmt.Errorf("%s is not managed by zick; rerun with --force to remove it", hookPath)
	}
	if err := os.Remove(hookPath); err != nil {
		return "", fmt.Errorf("remove %s: %w", hookPath, err)
	}
	return hookPath, nil
}

func preCommitPath(path string) (string, error) {
	root, gitPath, err := findGit(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", gitPath, err)
	}
	if info.IsDir() {
		return filepath.Join(gitPath, "hooks", "pre-commit"), nil
	}

	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", gitPath, err)
	}
	const prefix = "gitdir:"
	gitDir := strings.TrimSpace(string(data))
	if !strings.HasPrefix(gitDir, prefix) {
		return "", fmt.Errorf("%s is not a supported .git file", gitPath)
	}
	gitDir = strings.TrimSpace(strings.TrimPrefix(gitDir, prefix))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	return filepath.Join(gitDir, "hooks", "pre-commit"), nil
}

func findGit(path string) (string, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve hook path %s: %w", path, err)
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	for {
		gitPath := filepath.Join(abs, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return abs, gitPath, nil
		} else if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("stat %s: %w", gitPath, err)
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", "", fmt.Errorf("no .git directory found from %s", path)
		}
		abs = parent
	}
}

func isManaged(content string) bool {
	return strings.Contains(content, beginMarker) && strings.Contains(content, endMarker)
}

func script(opts InstallOptions) string {
	lines := []string{
		"#!/bin/sh",
		beginMarker,
		"set -eu",
		`repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`,
		`cd "$repo_root"`,
		"zick fresh .",
	}
	if opts.IncludeSecrets {
		lines = append(lines, fmt.Sprintf("zick secrets --tool %q .", opts.SecretsTool))
	}
	lines = append(lines, endMarker, "")
	return strings.Join(lines, "\n")
}

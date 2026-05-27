package tools

import (
	"os"
	"path/filepath"
)

type Betterleaks struct {
	Git bool
}

func NewBetterleaks(path string) *Betterleaks {
	return &Betterleaks{Git: isGitRepoRoot(path)}
}

func (b *Betterleaks) Name() string       { return "betterleaks" }
func (b *Betterleaks) BinaryName() string { return "betterleaks" }
func (b *Betterleaks) DockerImage() string {
	return "ghcr.io/betterleaks/betterleaks:latest"
}
func (b *Betterleaks) Args(path string) []string {
	if b.Git {
		return []string{"git", path}
	}
	return []string{"dir", path}
}

func isGitRepoRoot(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}

	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return true
	}
	return false
}

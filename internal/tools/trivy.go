package tools

import (
	"os"
	"path/filepath"
)

type Trivy struct {
	SARIFOutput string
}

func (t *Trivy) Name() string        { return "trivy" }
func (t *Trivy) BinaryName() string  { return "trivy" }
func (t *Trivy) DockerImage() string { return "aquasec/trivy:latest" }
func (t *Trivy) Args(path string) []string {
	args := []string{"fs", path}
	if t.SARIFOutput != "" {
		args = append(args, "--format", "sarif", "--output", t.SARIFOutput)
	}
	return args
}

// CacheMount persists the trivy vulnerability DB across Docker runs so it is
// not re-downloaded on every invocation.
func (t *Trivy) CacheMount() (string, string) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", ""
	}
	return filepath.Join(cacheDir, "trivy"), "/root/.cache/trivy"
}

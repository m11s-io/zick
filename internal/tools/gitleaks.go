package tools

type Gitleaks struct{}

func (g *Gitleaks) Name() string        { return "gitleaks" }
func (g *Gitleaks) BinaryName() string  { return "gitleaks" }
func (g *Gitleaks) DockerImage() string { return "ghcr.io/gitleaks/gitleaks:latest" }
func (g *Gitleaks) Args(path string) []string {
	return []string{"detect", "--source", path, "--no-banner"}
}

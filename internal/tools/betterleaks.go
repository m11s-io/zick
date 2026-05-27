package tools

type Betterleaks struct{}

func (b *Betterleaks) Name() string       { return "betterleaks" }
func (b *Betterleaks) BinaryName() string { return "betterleaks" }
func (b *Betterleaks) DockerImage() string {
	return "ghcr.io/betterleaks/betterleaks:latest"
}
func (b *Betterleaks) Args(path string) []string {
	return []string{"--path", path}
}

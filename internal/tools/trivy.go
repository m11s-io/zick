package tools

type Trivy struct{}

func (t *Trivy) Name() string        { return "trivy" }
func (t *Trivy) BinaryName() string  { return "trivy" }
func (t *Trivy) DockerImage() string { return "aquasec/trivy:latest" }
func (t *Trivy) Args(path string) []string {
	return []string{"fs", path}
}

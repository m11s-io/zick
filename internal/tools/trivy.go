package tools

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

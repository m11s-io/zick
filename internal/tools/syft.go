package tools

type Syft struct {
	Format string
	Output string
}

func (s *Syft) Name() string        { return "syft" }
func (s *Syft) BinaryName() string  { return "syft" }
func (s *Syft) DockerImage() string { return "ghcr.io/anchore/syft:latest" }
func (s *Syft) Args(path string) []string {
	format := s.Format
	if format == "" {
		format = "cyclonedx-json"
	}

	args := []string{path, "-o", format}
	if s.Output != "" {
		args = []string{path, "-o", format + "=" + s.Output}
	}
	return args
}

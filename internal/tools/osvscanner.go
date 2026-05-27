package tools

type OSVScanner struct {
	SARIFOutput string
}

func (o *OSVScanner) Name() string        { return "osv-scanner" }
func (o *OSVScanner) BinaryName() string  { return "osv-scanner" }
func (o *OSVScanner) DockerImage() string { return "ghcr.io/google/osv-scanner:latest" }
func (o *OSVScanner) Args(path string) []string {
	args := []string{"--recursive", path}
	if o.SARIFOutput != "" {
		args = append(args, "--format", "sarif", "--output", o.SARIFOutput)
	}
	return args
}

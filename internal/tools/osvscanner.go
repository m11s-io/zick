package tools

type OSVScanner struct{}

func (o *OSVScanner) Name() string        { return "osv-scanner" }
func (o *OSVScanner) BinaryName() string  { return "osv-scanner" }
func (o *OSVScanner) DockerImage() string { return "ghcr.io/google/osv-scanner:latest" }
func (o *OSVScanner) Args(path string) []string {
	return []string{"--recursive", path}
}

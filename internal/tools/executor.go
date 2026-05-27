package tools

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
)

// Tool describes a security tool that zick can orchestrate.
type Tool interface {
	Name() string
	BinaryName() string
	DockerImage() string
	Args(path string) []string
}

// Executor resolves tool execution: local binary first, Docker fallback.
// out and errOut mirror cobra's cmd.OutOrStdout() / cmd.ErrOrStderr() so
// output can be redirected in tests.
type Executor struct {
	out    io.Writer
	errOut io.Writer
}

func NewExecutor(out, errOut io.Writer) *Executor {
	return &Executor{out: out, errOut: errOut}
}

func (e *Executor) RunSecrets(path, toolName string) error {
	var t Tool

	switch toolName {
	case "betterleaks":
		t = &Betterleaks{}
	case "gitleaks":
		t = &Gitleaks{}
	default: // "auto"
		t = &Betterleaks{}
	}

	return e.run(t, path)
}

func (e *Executor) RunScan(path string, toolNames []string) error {
	for _, toolName := range toolNames {
		var t Tool
		switch toolName {
		case "osv-scanner":
			t = &OSVScanner{}
		case "trivy":
			t = &Trivy{}
		default:
			return fmt.Errorf("unsupported scanner %q", toolName)
		}

		fmt.Fprintf(e.out, "Running %s\n", t.Name())
		if err := e.run(t, path); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) run(t Tool, path string) error {
	if binary, err := exec.LookPath(t.BinaryName()); err == nil {
		return e.runLocal(binary, t.Args(path))
	}

	if _, err := exec.LookPath("docker"); err == nil {
		fmt.Fprintf(e.out, "%s not found in PATH — falling back to Docker (%s)\n", t.BinaryName(), t.DockerImage())
		return e.runDocker(t.DockerImage(), path, t.Args("/src"))
	}

	return fmt.Errorf("%s not found locally and Docker is not available.\nInstall %s or Docker to use this command", t.Name(), t.BinaryName())
}

func (e *Executor) runLocal(binary string, args []string) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdout = e.out
	cmd.Stderr = e.errOut
	return cmd.Run()
}

func (e *Executor) runDocker(image, hostPath string, args []string) error {
	absHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", hostPath, err)
	}

	dockerArgs := []string{
		"run", "--rm",
		"-v", absHostPath + ":/src",
		image,
	}
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = e.out
	cmd.Stderr = e.errOut
	return cmd.Run()
}

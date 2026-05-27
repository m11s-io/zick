package tools

import (
	"fmt"
	"io"
	"os/exec"
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
		return fmt.Errorf("gitleaks integration not yet implemented — coming in Stage 2")
	default: // "auto"
		t = &Betterleaks{}
	}

	return e.run(t, path)
}

func (e *Executor) run(t Tool, path string) error {
	if binary, err := exec.LookPath(t.BinaryName()); err == nil {
		return e.runLocal(binary, t.Args(path))
	}

	if _, err := exec.LookPath("docker"); err == nil {
		fmt.Fprintf(e.out, "%s not found in PATH — falling back to Docker (%s)\n", t.BinaryName(), t.DockerImage())
		return e.runDocker(t.DockerImage(), path, t.Args("."))
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
	dockerArgs := []string{
		"run", "--rm",
		"-v", hostPath + ":/src",
		image,
	}
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = e.out
	cmd.Stderr = e.errOut
	return cmd.Run()
}

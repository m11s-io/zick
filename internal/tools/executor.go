package tools

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/m11s-io/zick/internal/cli"
)

const imagePullInterval = 7 * 24 * time.Hour

// Tool describes a security tool that zick can orchestrate.
type Tool interface {
	Name() string
	BinaryName() string
	DockerImage() string
	Args(path string) []string
}

// dockerCacher is an optional interface for tools that need a persistent cache
// directory mounted into the Docker container (e.g. trivy vulnerability DB).
type dockerCacher interface {
	CacheMount() (hostDir, containerDir string)
}

type ScanOptions struct {
	SARIFOutput string
}

type SBOMOptions struct {
	Format string
	Output string
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
	switch toolName {
	case "betterleaks":
		return e.run(NewBetterleaks(path), path)
	case "gitleaks":
		return e.run(&Gitleaks{}, path)
	default: // "auto"
		return e.run(NewBetterleaks(path), path)
	}
}

func (e *Executor) RunScan(path string, toolNames []string, opts ScanOptions) error {
	for _, toolName := range toolNames {
		var t Tool
		switch toolName {
		case "osv-scanner":
			t = &OSVScanner{SARIFOutput: opts.SARIFOutput}
		case "trivy":
			t = &Trivy{SARIFOutput: opts.SARIFOutput}
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

func (e *Executor) RunSBOM(path string, opts SBOMOptions) error {
	return e.run(&Syft{Format: opts.Format, Output: opts.Output}, path)
}

func (e *Executor) run(t Tool, path string) error {
	if binary, err := exec.LookPath(t.BinaryName()); err == nil {
		return e.runLocal(binary, t.Args(path))
	}

	if _, err := exec.LookPath("docker"); err == nil {
		absPath, _ := filepath.Abs(path)
		fmt.Fprintf(e.errOut, "%s not found in PATH — falling back to Docker (%s)\n", t.BinaryName(), t.DockerImage())
		fmt.Fprintf(e.errOut, "mounting %s → /src\n", absPath)
		e.pullIfStale(t.DockerImage())
		var cacheMount [2]string
		if dc, ok := t.(dockerCacher); ok {
			cacheMount[0], cacheMount[1] = dc.CacheMount()
		}
		return e.runDocker(t.DockerImage(), path, cacheMount, t.Args("."))
	}

	return fmt.Errorf("%s not found locally and Docker is not available.\nInstall %s or Docker to use this command", t.Name(), t.BinaryName())
}

// pullIfStale pulls the Docker image if it hasn't been pulled in the last 7
// days. The timestamp is tracked in ~/.cache/zick/. A failed pull is silently
// ignored so offline use continues to work with the cached image.
func (e *Executor) pullIfStale(image string) {
	tsFile, err := imagePullTimestampPath(image)
	if err != nil {
		return // can't determine cache dir; skip pull silently
	}

	if info, err := os.Stat(tsFile); err == nil && time.Since(info.ModTime()) < imagePullInterval {
		return
	}

	fmt.Fprintf(e.errOut, "pulling %s...\n", image)
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = e.errOut
	cmd.Stderr = e.errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(e.errOut, "warning: could not pull %s: %v\n", image, err)
		return
	}

	_ = os.MkdirAll(filepath.Dir(tsFile), 0o755)
	_ = os.WriteFile(tsFile, []byte{}, 0o644)
}

func imagePullTimestampPath(image string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	safe := strings.NewReplacer("/", "_", ":", "_").Replace(image)
	return filepath.Join(cacheDir, "zick", safe+"-last-pull"), nil
}

func (e *Executor) runLocal(binary string, args []string) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdout = e.out
	cmd.Stderr = e.errOut
	return silentExit(cmd.Run())
}

func (e *Executor) runDocker(image, hostPath string, cacheMount [2]string, args []string) error {
	absHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", hostPath, err)
	}

	dockerArgs := []string{
		"run", "--rm",
		"-e", "GIT_CONFIG_COUNT=1",
		"-e", "GIT_CONFIG_KEY_0=safe.directory",
		"-e", "GIT_CONFIG_VALUE_0=/src",
		"-v", absHostPath + ":/src",
		"-w", "/src",
	}

	if cacheMount[0] != "" {
		if err := os.MkdirAll(cacheMount[0], 0o755); err == nil {
			dockerArgs = append(dockerArgs, "-v", cacheMount[0]+":"+cacheMount[1])
		}
	}

	dockerArgs = append(dockerArgs, image)
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = e.out
	cmd.Stderr = e.errOut
	return silentExit(cmd.Run())
}

func silentExit(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &cli.SilentError{Code: exitErr.ExitCode()}
	}
	return err
}

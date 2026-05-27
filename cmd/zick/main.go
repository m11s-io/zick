package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/m11s-io/zick/internal/cli"
	"github.com/spf13/cobra"
)

// version, commit, date are set at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "zick",
		Short:        "Developer-first security scanning CLI",
		Long:         "zick orchestrates best-in-class open-source security tools into a single binary.\nRun it locally, in Docker, or point it at a deployed cluster service.",
		Version:      fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Command groups give the help output clear sections as the command set grows.
	root.AddGroup(
		&cobra.Group{ID: "scan", Title: "Scanning:"},
	)

	root.AddCommand(
		newFreshCmd(),
		newSecretsCmd(),
	)

	return root
}

func main() {
	root := newRootCmd()
	if err := root.ExecuteContext(context.Background()); err != nil {
		var se *cli.SilentError
		if errors.As(err, &se) {
			os.Exit(se.Code)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

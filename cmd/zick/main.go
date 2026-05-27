package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/m11s-io/zick/internal/app"
	"github.com/m11s-io/zick/internal/cli"
)

// version, commit, date are set at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ver := fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
	root := app.NewRootCmd(ver)
	if err := root.ExecuteContext(context.Background()); err != nil {
		var se *cli.SilentError
		if errors.As(err, &se) {
			os.Exit(se.Code)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

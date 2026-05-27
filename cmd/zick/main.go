package main

import (
	"context"
	"errors"
	"io"
	"os"

	"charm.land/fang/v2"
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
	root := app.NewRootCmd("")
	err := fang.Execute(context.Background(), root,
		fang.WithVersion(version+" (built "+date+")"),
		fang.WithCommit(commit),
		fang.WithErrorHandler(func(w io.Writer, styles fang.Styles, err error) {
			var se *cli.SilentError
			if !errors.As(err, &se) {
				fang.DefaultErrorHandler(w, styles, err)
			}
		}),
	)
	if err != nil {
		var se *cli.SilentError
		if errors.As(err, &se) {
			os.Exit(se.Code)
		}
		os.Exit(1)
	}
}

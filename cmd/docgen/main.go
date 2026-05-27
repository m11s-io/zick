package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/m11s-io/zick/internal/app"
	"github.com/spf13/cobra/doc"
)

func main() {
	out := flag.String("out", "./docs/cli", "output directory")
	front := flag.Bool("frontmatter", false, "prepend YAML front matter for MkDocs")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := app.NewRootCmd("dev")
	root.DisableAutoGenTag = true

	if *front {
		prep := func(filename string) string {
			base := filepath.Base(filename)
			name := strings.TrimSuffix(base, filepath.Ext(base))
			title := strings.ReplaceAll(name, "_", " ")
			return fmt.Sprintf("---\ntitle: %q\nslug: %q\ndescription: \"CLI reference for %s\"\n---\n\n", title, name, title)
		}
		link := func(name string) string { return strings.ToLower(name) }
		if err := doc.GenMarkdownTreeCustom(root, *out, prep, link); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := doc.GenMarkdownTree(root, *out); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("Docs written to %s\n", *out)
}

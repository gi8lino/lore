package main

import (
	"context"
	"os"

	"github.com/gi8lino/lore/internal/cli"
	"github.com/gi8lino/lore/web"
)

var (
	Version = "dev"
	Commit  = "none"
)

// main runs the Lore command-line application and exits non-zero on failure.
func main() {
	if err := cli.Run(
		context.Background(),
		os.Args[1:],
		web.Assets,
		Version,
		Commit,
		os.Stdout,
		os.Stderr,
	); err != nil {
		os.Exit(1)
	}
}

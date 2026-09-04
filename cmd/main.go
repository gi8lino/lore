package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gi8lino/lore/internal/app"
	"github.com/gi8lino/lore/internal/site"
	"github.com/gi8lino/lore/web"
)

var (
	Version = "dev"
	Commit  = "none"
)

// main starts Lore or executes a build-time command and exits non-zero on failure.
func main() {
	ctx := context.Background()
	args := os.Args[1:]
	var err error
	if len(args) > 0 && args[0] == "site" {
		err = site.Run(ctx, web.Assets, Version, Commit, args[1:], os.Stdout, os.Stderr)
	} else {
		err = app.Run(ctx, web.Assets, Version, Commit, args, os.Stdout, os.Stderr)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Package cli defines Lore's command tree and binds command-line flags to application operations.
package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/mirror"
	"github.com/gi8lino/lore/internal/serve"
	"github.com/gi8lino/lore/internal/site"
)

// Run parses and executes one Lore command.
func Run(
	ctx context.Context,
	args []string,
	appFS fs.FS,
	version, commit string,
	stdout, stderr io.Writer,
) error {
	root := tinyflags.NewCommand("lore", tinyflags.ContinueOnError).RequireCommand()

	root.Version(version)

	serveCommand := root.Command("serve", "Run the Lore server")
	resolveServeConfig := serve.BindFlags(serveCommand.FlagSet)

	serveCommand.Run(func(ctx context.Context) error {
		return serve.Run(
			ctx,
			appFS,
			resolveServeConfig(),
			serveCommand.OverriddenValues(),
			version,
			commit,
			stdout,
		)
	})

	buildCommand := root.Command("build", "Build a read-only static documentation site")
	resolveBuildConfig := site.BindFlags(buildCommand.FlagSet)

	buildCommand.Run(func(ctx context.Context) error {
		cfg, err := resolveBuildConfig()
		if err != nil {
			return err
		}

		return site.Run(
			ctx,
			appFS,
			cfg,
			buildCommand.OverriddenValues(),
			stdout,
		)
	})

	mirrorCommand := root.Command("mirror", "Export PostgreSQL content as a Git-friendly Markdown mirror")
	resolveMirrorConfig := mirror.BindFlags(mirrorCommand.FlagSet)

	mirrorCommand.Run(func(ctx context.Context) error {
		return mirror.Run(ctx, resolveMirrorConfig(), stdout)
	})

	runner, err := root.ParseRunner(args)
	if err != nil {
		switch {
		case tinyflags.IsHelpRequested(err), tinyflags.IsVersionRequested(err):
			_, _ = fmt.Fprint(stdout, err.Error())
			return nil
		case tinyflags.IsCommandRequired(err):
			help, _ := tinyflags.HelpText(err)
			_, _ = fmt.Fprint(stderr, help)

			return nil
		default:
			_, _ = fmt.Fprintln(stderr, err)
			return err
		}
	}

	return runner.Run(ctx)
}

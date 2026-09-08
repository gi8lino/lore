// Package cli defines Lore's command tree and binds command-line flags to application operations.
package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/app"
	"github.com/gi8lino/lore/internal/config"
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

	serve := root.Command("serve", "Run the Lore server")
	serveConfig := config.BindFlags(serve.FlagSet)

	serve.Run(func(ctx context.Context) error {
		cfg := serveConfig()
		return app.Run(
			ctx,
			appFS,
			cfg.ListenAddress,
			cfg.DatabaseURL,
			cfg.PublicURL,
			cfg.PDFURL,
			cfg.AuthModeOverride,
			cfg.TrustedUsernameHeaders,
			cfg.TrustedEmailHeaders,
			cfg.TrustedDisplayNameHeaders,
			cfg.OIDCIssuer,
			cfg.OIDCClientID,
			cfg.OIDCClientSecret,
			cfg.SessionSecret,
			cfg.LocalLogin,
			cfg.ThemeDirectory,
			cfg.LogFormat,
			cfg.Debug,
			cfg.AccessLog,
			serve.OverriddenValues(),
			version,
			commit,
			stdout,
		)
	})

	build := root.Command("build", "Build a read-only static documentation site")
	buildConfig := site.BindFlags(build.FlagSet)

	build.Run(func(ctx context.Context) error {
		cfg, err := buildConfig()
		if err != nil {
			return err
		}

		return site.Run(
			ctx,
			appFS,
			cfg,
			build.OverriddenValues(),
			stdout,
		)
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
			return err
		}
	}

	return runner.Run(ctx)
}

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
	buildConfig := bindBuildFlags(build.FlagSet)

	build.Run(func(ctx context.Context) error {
		cfg, err := buildConfig()
		if err != nil {
			return err
		}

		return site.Build(
			ctx,
			appFS,
			version,
			commit,
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

func bindBuildFlags(flags *tinyflags.FlagSet) func() (site.Config, error) {
	defaults := site.DefaultConfig()

	configPath := flags.String("config", site.DefaultConfigPath, "TOML site configuration file").
		Placeholder("FILE")
	siteName := flags.String("site-name", defaults.SiteName, "Site title").
		Placeholder("NAME")
	siteURL := flags.String("site-url", defaults.SiteURL, "Published site URL").
		Placeholder("URL")
	source := flags.String("source", defaults.SourceDir, "Markdown source directory").
		Placeholder("DIR")
	output := flags.String("output", defaults.OutputDir, "Generated site directory").
		Placeholder("DIR")
	theme := flags.String("theme", defaults.Theme, "Lore theme").
		Placeholder("THEME")
	language := flags.String("language", defaults.Language, "HTML content language").
		Placeholder("LANG")
	mermaid := flags.Bool("mermaid", defaults.Mermaid, "Enable Mermaid rendering").Strict()

	return func() (site.Config, error) {
		cfg, err := site.LoadConfig(*configPath.Value(), configPath.Changed())
		if err != nil {
			return site.Config{}, err
		}

		if siteName.Changed() {
			cfg.SiteName = *siteName.Value()
		}
		if siteURL.Changed() {
			cfg.SiteURL = *siteURL.Value()
		}
		if source.Changed() {
			cfg.SourceDir = *source.Value()
		}
		if output.Changed() {
			cfg.OutputDir = *output.Value()
		}
		if theme.Changed() {
			cfg.Theme = *theme.Value()
		}
		if language.Changed() {
			cfg.Language = *language.Value()
		}
		if mermaid.Changed() {
			cfg.Mermaid = *mermaid.Value()
		}

		return cfg, nil
	}
}

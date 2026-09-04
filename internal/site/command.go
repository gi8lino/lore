package site

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// Run executes the filesystem-backed static site command.
func Run(
	ctx context.Context,
	appFS fs.FS,
	version, commit string,
	args []string,
	stdout, stderr io.Writer,
) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		writeUsage(stdout)
		return nil
	}
	if args[0] != "build" {
		return fmt.Errorf("unknown site command %q", args[0])
	}
	return runBuild(ctx, appFS, version, commit, args[1:], stdout, stderr)
}

func runBuild(
	ctx context.Context,
	appFS fs.FS,
	version, commit string,
	args []string,
	stdout, stderr io.Writer,
) error {
	configPath, explicitlyConfigured := configPathFromArgs(args)
	config, err := loadConfig(configPath, explicitlyConfigured)
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("lore site build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configPath, "config", configPath, "TOML site configuration file")
	flags.StringVar(&config.SiteName, "site-name", config.SiteName, "site title")
	flags.StringVar(&config.SiteURL, "site-url", config.SiteURL, "published site URL")
	flags.StringVar(&config.SourceDir, "source", config.SourceDir, "Markdown source directory")
	flags.StringVar(&config.OutputDir, "output", config.OutputDir, "generated site directory")
	flags.StringVar(&config.Theme, "theme", config.Theme, "Lore theme")
	flags.StringVar(&config.Language, "language", config.Language, "HTML content language")
	flags.BoolVar(&config.Mermaid, "mermaid", config.Mermaid, "enable Mermaid rendering")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected site build arguments: %s", strings.Join(flags.Args(), " "))
	}
	if err := config.validate(); err != nil {
		return err
	}

	builder := NewBuilder(appFS, version, commit)
	result, err := builder.Build(ctx, config)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir)
	return nil
}

func configPathFromArgs(args []string) (string, bool) {
	for index, arg := range args {
		if (arg == "--config" || arg == "-config") && index+1 < len(args) {
			return args[index+1], true
		}
		for _, prefix := range []string{"--config=", "-config="} {
			if value, ok := strings.CutPrefix(arg, prefix); ok {
				return value, true
			}
		}
	}
	return defaultConfigPath, false
}

func writeUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  lore site build [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Build a read-only Lore documentation site from Markdown files without PostgreSQL.")
	_, _ = fmt.Fprintln(w, "The default configuration file is lore-site.toml; it is optional.")
}

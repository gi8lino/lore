package site

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/gi8lino/lore/internal/logging"
)

// Run builds one filesystem-backed static documentation site.
func Run(
	ctx context.Context,
	appFS fs.FS,
	version, commit string,
	cfg Config,
	overrides map[string]any,
	stdout io.Writer,
) error {
	logger := logging.Setup(cfg.LogFormat, false, stdout)
	setupLogger := logger.With("component", "setup")

	if len(overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", overrides,
		)
	}

	result, err := NewBuilder(appFS, version, commit).Build(ctx, cfg)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir)

	return nil
}

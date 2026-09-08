package site

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/gi8lino/lore/internal/logging"
)

// Run builds one filesystem-backed static documentation site.
func Run(ctx context.Context, appFS fs.FS, config Config, overrides map[string]any, stdout io.Writer) error {
	logger := logging.Setup(config.logFormat, false, stdout)
	setupLogger := logger.With("component", "setup")

	if len(overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", overrides,
		)
	}

	result, err := newBuilder(appFS).build(ctx, config)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.pages, result.outputDir)
	return nil
}

package site

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/gi8lino/lore/internal/logging"
)

// Build generates one filesystem-backed static documentation site.
func Build(
	ctx context.Context,
	appFS fs.FS,
	version, commit string,
	config Config,
	overrides map[string]any,
	stdout io.Writer,
) error {
	setupLogger := logging.Setup(logging.LogFormatText, false, stdout).With("component", "setup")

	if len(overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", overrides,
		)
	}

	result, err := NewBuilder(appFS, version, commit).Build(ctx, config)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir)

	return nil
}

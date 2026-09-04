package site

import (
	"context"
	"fmt"
	"io"
	"io/fs"
)

// Build generates one filesystem-backed static documentation site.
func Build(
	ctx context.Context,
	appFS fs.FS,
	version, commit string,
	config Config,
	stdout io.Writer,
) error {
	result, err := NewBuilder(appFS, version, commit).Build(ctx, config)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Built %d pages into %s\n", result.Pages, result.OutputDir)
	return nil
}

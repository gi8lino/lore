// Package bundled supplies distribution bytes, not a separate plugin runtime.
package bundled

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/gi8lino/lore/internal/plugin"
)

//go:embed *.loreplugin
var Packages embed.FS

func Load(ctx context.Context, manager *plugin.Manager) error {
	names, err := fs.Glob(Packages, "*.loreplugin")
	if err != nil {
		return err
	}
	for _, name := range names {
		data, err := Packages.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := manager.Load(ctx, data, plugin.SourceBundled); err != nil {
			return fmt.Errorf("load bundled package %s: %w", name, err)
		}
	}
	return nil
}

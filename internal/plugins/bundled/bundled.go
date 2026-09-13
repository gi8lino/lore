// Package bundled supplies distribution bytes, not a separate plugin runtime.
package bundled

import (
	"context"
	"embed"
	"io/fs"

	"github.com/gi8lino/lore/internal/plugin"
)

//go:embed *.loreplugin
var Packages embed.FS

// Load returns embedded bundled plugin archives in deterministic filename order.
func Load(ctx context.Context, manager *plugin.Manager) error {
	names, err := fs.Glob(Packages, "*.loreplugin")
	if err != nil {
		return err
	}
	archives := make([][]byte, 0, len(names))
	for _, name := range names {
		data, err := Packages.ReadFile(name)
		if err != nil {
			return err
		}
		archives = append(archives, data)
	}
	return manager.Bootstrap(ctx, archives)
}

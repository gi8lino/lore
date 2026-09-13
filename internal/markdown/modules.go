package markdown

import (
	"context"
	"github.com/gi8lino/lore/internal/pagereport"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/internal/plugins/bundled"
	"github.com/gi8lino/lore/internal/subpages"
)

// New constructs a renderer and owns its plugin runtime. Server and static
// build callers close it when their application scope ends.
func New(ctx context.Context) (*Renderer, error) {
	registry := coreRegistry()
	runtime, err := wasm.New(ctx, wasm.Limits{})
	if err != nil {
		return nil, err
	}
	manager := plugin.NewManager(registry, runtime)
	if err := bundled.Load(ctx, manager); err != nil {
		_ = manager.Close(context.Background())
		return nil, err
	}
	renderer := NewWithRegistry(registry)
	renderer.manager = manager
	return renderer, nil
}

// coreRegistry keeps the native macro adapters until the capability migration.
func coreRegistry() *plugin.Registry {
	registry := &plugin.Registry{}
	register := func(d plugin.Descriptor, c plugin.Contributions) {
		if err := registry.Register(d, c); err != nil {
			panic(err)
		}
	}
	register(plugin.Descriptor{ID: "io.lore.subpages", Name: "Subpages", DefaultEnabled: true}, plugin.Contributions{
		Macros: []plugin.Macro{plugin.BoundMacro[subpages.Options]{MacroName: "subpages", ParseOptions: subpages.Parse, EmptyWhenUnbound: true}},
	})
	register(plugin.Descriptor{ID: "io.lore.page-report", Name: "Page Report", DefaultEnabled: true}, plugin.Contributions{
		Macros: []plugin.Macro{plugin.BoundMacro[pagereport.Options]{MacroName: "pages", ParseOptions: pagereport.Parse}},
	})
	return registry
}

// moduleFeatures translates legacy settings at the composition boundary. It
// does not activate modules absent from the registry.
func moduleFeatures(options Options) map[string]bool {
	return map[string]bool{"io.lore.callouts": options.Callouts}
}

package markdown

import (
	"context"
	"maps"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/internal/plugins/bundled"
)

// New constructs a renderer and owns its plugin runtime. Server and static
// build callers close it when their application scope ends.
func New(ctx context.Context, runtimeOptions ...wasm.Option) (*Renderer, error) {
	return NewWithPluginStore(ctx, nil, runtimeOptions...)
}

// NewWithPluginStore restores durable plugin lifecycle state before rendering.
func NewWithPluginStore(ctx context.Context, store plugin.Store, runtimeOptions ...wasm.Option) (*Renderer, error) {
	registry := &plugin.Registry{}
	options := []wasm.Option{wasm.WithPermissions("pages:read", "browser:render")}
	options = append(options, runtimeOptions...)
	runtime, err := wasm.New(ctx, wasm.Limits{}, options...)
	if err != nil {
		return nil, err
	}
	managerOptions := []plugin.ManagerOption{}
	if store != nil {
		managerOptions = append(managerOptions, plugin.WithStore(store))
		if values, ok := store.(plugin.Storage); ok {
			managerOptions = append(managerOptions, plugin.WithStorage(values))
		}
	}
	manager := plugin.NewManager(registry, runtime, managerOptions...)
	if err := bundled.Load(ctx, manager); err != nil {
		_ = manager.Close(context.Background())
		return nil, err
	}
	renderer := NewWithRegistry(registry)
	renderer.manager = manager
	return renderer, nil
}

// moduleFeatures translates legacy settings at the composition boundary. It
// does not activate modules absent from the registry.
func moduleFeatures(options Options) map[string]bool {
	return map[string]bool{"io.lore.callouts": options.Callouts, "io.lore.mermaid": options.Mermaid, "io.lore.tables": options.Tables, "io.lore.tables.tables": options.Tables, "io.lore.tables.styles": options.TableStyles, "io.lore.tables.sorting": options.TableSorting, "io.lore.tables.filtering": options.TableFiltering}
}

// pluginFeatures overlays persisted plugin-owned settings on legacy request options.
func (r *Renderer) pluginFeatures(options Options) map[string]bool {
	features := moduleFeatures(options)
	if r.manager != nil {
		maps.Copy(features, r.manager.FeatureSettings())
	}
	return features
}

// PluginManager exposes lifecycle operations to the trusted application layer.
// Renderers built from an external registry have no owned manager.
func (r *Renderer) PluginManager() *plugin.Manager { return r.manager }

// ValidateFeatures checks declarative plugin settings dependencies.
func (r *Renderer) ValidateFeatures(options Options) error {
	return r.registry.Snapshot().ValidateFeatures(moduleFeatures(options))
}

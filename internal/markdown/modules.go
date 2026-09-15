package markdown

import (
	"context"

	"github.com/gi8lino/lore/internal/firstparty"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
)

// New constructs a renderer and owns its plugin runtime. Server and static
// build callers close it when their application scope ends.
func New(ctx context.Context, runtimeOptions ...wasm.Option) (*Renderer, error) {
	return NewWithPluginStore(ctx, nil, runtimeOptions...)
}

// NewWithPluginStore restores durable plugin lifecycle state before rendering.
func NewWithPluginStore(ctx context.Context, store plugin.Store, runtimeOptions ...wasm.Option) (*Renderer, error) {
	registry := &plugin.Registry{}
	options := []wasm.Option{wasm.WithPermissions("pages:read", "pages:content", "browser:render")}
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
	if err := firstparty.Load(ctx, manager); err != nil {
		_ = manager.Close(context.Background())
		return nil, err
	}
	return NewWithManager(registry, manager), nil
}

// pluginFeatures returns lifecycle and plugin-owned settings for the current renderer.
func (r *Renderer) pluginFeatures() map[string]bool {
	if r.manager == nil {
		return nil
	}
	return r.manager.FeatureSettings()
}

// PluginManager exposes lifecycle operations to the trusted application layer.
// Renderers built from an external registry have no owned manager.
func (r *Renderer) PluginManager() *plugin.Manager { return r.manager }

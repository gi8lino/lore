package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gi8lino/lore/internal/pluginpackage"
)

// Runtime loads validated packages without depending on their distribution.
type Runtime interface {
	Load(context.Context, *pluginpackage.Package) (Instance, error)
	Close(context.Context) error
}

// Instance owns executable contributions and releases their runtime resources.
type Instance interface {
	Contributions() Contributions
	Close(context.Context) error
}

// Source is descriptive metadata only; it never grants a runtime capability.
type Source string

const (
	SourceBundled   Source = "bundled"
	SourceInstalled Source = "installed"
)

type LoadedPlugin struct {
	Manifest pluginpackage.Manifest
	Source   Source
	Digest   [32]byte
}

type managedPlugin struct {
	metadata LoadedPlugin
	instance Instance
}

// Manager coordinates package validation, runtime ownership, and atomic registry
// publication. Persistence and user-facing installation belong to later phases.
type Manager struct {
	mu       sync.Mutex
	registry *Registry
	runtime  Runtime
	loaded   map[string]managedPlugin
	order    []string
	closed   bool
}

func NewManager(registry *Registry, runtime Runtime) *Manager {
	return &Manager{registry: registry, runtime: runtime, loaded: make(map[string]managedPlugin)}
}

// Load uses the same package reader and runtime for all sources. A failed load
// leaves the active registry unchanged and closes any newly created instance.
func (m *Manager) Load(ctx context.Context, archive []byte, source Source) (LoadedPlugin, error) {
	pkg, err := pluginpackage.Read(archive)
	if err != nil {
		return LoadedPlugin{}, err
	}
	manifest := pkg.Manifest()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return LoadedPlugin{}, errors.New("plugin manager is closed")
	}
	if source != SourceBundled && source != SourceInstalled {
		return LoadedPlugin{}, errors.New("invalid plugin source")
	}
	if _, exists := m.loaded[manifest.ID]; exists {
		return LoadedPlugin{}, fmt.Errorf("plugin %s is already loaded", manifest.ID)
	}
	instance, err := m.runtime.Load(ctx, pkg)
	if err != nil {
		return LoadedPlugin{}, fmt.Errorf("load plugin %s: %w", manifest.ID, err)
	}
	descriptor := Descriptor{ID: manifest.ID, Name: manifest.Name, Description: manifest.Description, DefaultEnabled: manifest.DefaultEnabled, Requires: manifest.Requires}
	if err := m.registry.Register(descriptor, instance.Contributions()); err != nil {
		_ = instance.Close(context.Background())
		return LoadedPlugin{}, err
	}
	metadata := LoadedPlugin{Manifest: manifest, Source: source, Digest: pkg.Digest()}
	m.loaded[manifest.ID] = managedPlugin{metadata, instance}
	m.order = append(m.order, manifest.ID)
	return cloneLoaded(metadata), nil
}

// Unload unregisters before closing the instance. In-flight calls drain; a
// stale snapshot that has not entered the instance receives a closed error.
func (m *Manager) Unload(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unload(ctx, id)
}
func (m *Manager) unload(ctx context.Context, id string) error {
	loaded, ok := m.loaded[id]
	if !ok {
		return fmt.Errorf("plugin %s is not loaded", id)
	}
	if err := m.registry.Unregister(id); err != nil {
		return err
	}
	delete(m.loaded, id)
	return loaded.instance.Close(ctx)
}

func (m *Manager) Plugins() []LoadedPlugin {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]LoadedPlugin, 0, len(m.loaded))
	seen := make(map[string]bool)
	for _, id := range m.order {
		if loaded, ok := m.loaded[id]; ok && !seen[id] {
			result = append(result, cloneLoaded(loaded.metadata))
			seen[id] = true
		}
	}
	return result
}

func cloneLoaded(metadata LoadedPlugin) LoadedPlugin {
	metadata.Manifest.Modules = append([]pluginpackage.Module(nil), metadata.Manifest.Modules...)
	metadata.Manifest.Requires = append([]string(nil), metadata.Manifest.Requires...)
	metadata.Manifest.Permissions = append([]string(nil), metadata.Manifest.Permissions...)
	return metadata
}

func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	var result error
	for index := len(m.order) - 1; index >= 0; index-- {
		id := m.order[index]
		if loaded, ok := m.loaded[id]; ok {
			result = errors.Join(result, m.registry.Unregister(id), loaded.instance.Close(ctx))
			delete(m.loaded, id)
		}
	}
	return errors.Join(result, m.runtime.Close(ctx))
}

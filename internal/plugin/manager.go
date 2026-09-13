package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	Enabled  bool
	Manifest pluginpackage.Manifest
	Source   Source
	Digest   [32]byte
}

type managedPlugin struct {
	archive  []byte
	metadata LoadedPlugin
	instance Instance
}

// Manager coordinates package validation, runtime ownership, and atomic registry
// publication. Durable lifecycle state is provided by a small store interface.
type Manager struct {
	mu          sync.Mutex
	registry    *Registry
	runtime     Runtime
	loaded      map[string]managedPlugin
	order       []string
	closed      bool
	store       Store
	bundled     map[string][]byte
	retirements []*retirement
}

func NewManager(registry *Registry, runtime Runtime, options ...ManagerOption) *Manager {
	m := &Manager{registry: registry, runtime: runtime, loaded: make(map[string]managedPlugin), bundled: make(map[string][]byte), store: &memoryStore{records: make(map[string]Record)}}
	for _, option := range options {
		option(m)
	}
	return m
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
	metadata := LoadedPlugin{Manifest: manifest, Source: source, Digest: pkg.Digest(), Enabled: true}
	m.loaded[manifest.ID] = managedPlugin{metadata: metadata, instance: instance, archive: append([]byte(nil), archive...)}
	m.order = append(m.order, manifest.ID)
	if source == SourceBundled {
		m.bundled[manifest.ID] = append([]byte(nil), archive...)
	}
	return cloneLoaded(metadata), nil
}

// Unload removes a transient entry. Acquired render snapshots retain the old
// instance until release; unleased inspection snapshots carry no such guarantee.
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
	if loaded.instance != nil {
		old, err := m.registry.transition(id, nil, true, nil)
		if err != nil {
			return err
		}
		m.retire(old, loaded.instance)
	}
	delete(m.loaded, id)
	m.order = slices.DeleteFunc(m.order, func(value string) bool { return value == id })
	return nil
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
	if !m.closed {
		m.closed = true

		lives := m.registry.detach(m.loaded)
		for id, loaded := range m.loaded {
			if loaded.instance != nil {
				m.retire(lives[id], loaded.instance)
			}
			delete(m.loaded, id)
		}
		m.order = nil
	}
	pending := append([]*retirement(nil), m.retirements...)
	m.mu.Unlock()
	var result error
	for _, retired := range pending {
		select {
		case <-retired.done:
			result = errors.Join(result, retired.err)
		case <-ctx.Done():
			return errors.Join(result, ctx.Err())
		}
	}
	return errors.Join(result, m.runtime.Close(ctx))
}

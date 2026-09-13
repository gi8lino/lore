package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/gi8lino/lore/internal/pluginpackage"
)

type retirement struct {
	done chan struct{}
	err  error
}

func (m *Manager) retire(life *lifetime, instance Instance) {
	// Drop completed successful retirements to keep long-running managers bounded.
	m.retirements = slices.DeleteFunc(m.retirements, func(r *retirement) bool {
		select {
		case <-r.done:
			return r.err == nil
		default:
			return false
		}
	})
	retired := &retirement{done: make(chan struct{})}
	m.retirements = append(m.retirements, retired)
	closeInstance := func() { retired.err = instance.Close(context.Background()); close(retired.done) }
	if life == nil {
		closeInstance()
		return
	}
	drained := life.retire()
	select {
	case <-drained:
		closeInstance()
	default:
		go func() { <-drained; closeInstance() }()
	}
}
func descriptor(manifest pluginpackage.Manifest) Descriptor {
	return Descriptor{ID: manifest.ID, Name: manifest.Name, Description: manifest.Description, DefaultEnabled: manifest.DefaultEnabled, Requires: manifest.Requires}
}
func recordFor(item managedPlugin) Record {
	record := Record{ID: item.metadata.Manifest.ID, Source: item.metadata.Source, Enabled: item.metadata.Enabled}
	if record.Source == SourceInstalled {
		record.Package = item.archive
	}
	return record
}

// Install validates and starts a package before durable, atomic publication.
func (m *Manager) Install(ctx context.Context, archive []byte) (LoadedPlugin, error) {
	pkg, err := pluginpackage.Read(archive)
	if err != nil {
		return LoadedPlugin{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return LoadedPlugin{}, errors.New("plugin manager is closed")
	}
	id := pkg.Manifest().ID
	if _, ok := m.loaded[id]; ok {
		return LoadedPlugin{}, fmt.Errorf("plugin %s is already installed", id)
	}
	item, err := m.prepare(ctx, pkg, archive, SourceInstalled, true)
	if err != nil {
		return LoadedPlugin{}, err
	}
	if err = m.publish(ctx, id, item); err != nil {
		return LoadedPlugin{}, err
	}
	m.order = append(m.order, id)
	return cloneLoaded(item.metadata), nil
}
func (m *Manager) prepare(ctx context.Context, pkg *pluginpackage.Package, archive []byte, source Source, enabled bool) (managedPlugin, error) {
	// Even disabled upgrades validate executable compatibility before persistence.
	instance, err := m.runtime.Load(ctx, pkg)
	if err != nil {
		return managedPlugin{}, err
	}
	item := managedPlugin{metadata: LoadedPlugin{Manifest: pkg.Manifest(), Source: source, Digest: pkg.Digest(), Enabled: enabled}, archive: append([]byte(nil), archive...), instance: instance}
	if !enabled {
		if err := instance.Close(ctx); err != nil {
			return managedPlugin{}, err
		}
		item.instance = nil
	}
	return item, nil
}
func (m *Manager) publish(ctx context.Context, id string, item managedPlugin) error {
	previous := m.loaded[id]
	commit := func() error { return m.store.SavePlugin(ctx, recordFor(item)) }
	var old *lifetime
	var err error
	if item.instance != nil {
		entry := &Entry{Descriptor: descriptor(item.metadata.Manifest), Contributions: item.instance.Contributions()}
		old, err = m.registry.transition(id, entry, previous.instance != nil, commit)
	} else if previous.instance != nil {
		old, err = m.registry.transition(id, nil, true, commit)
	} else {
		err = commit()
	}
	if err != nil {
		if item.instance != nil {
			_ = item.instance.Close(context.Background())
		}
		return err
	}
	m.loaded[id] = item
	if previous.instance != nil {
		m.retire(old, previous.instance)
	}
	return nil
}
func (m *Manager) Enable(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, err := m.find(id)
	if err != nil {
		return err
	}
	if item.metadata.Enabled {
		return nil
	}
	pkg, err := pluginpackage.Read(item.archive)
	if err != nil {
		return err
	}
	prepared, err := m.prepare(ctx, pkg, item.archive, item.metadata.Source, true)
	if err != nil {
		return err
	}
	return m.publish(ctx, id, prepared)
}
func (m *Manager) Disable(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, err := m.find(id)
	if err != nil {
		return err
	}
	if !item.metadata.Enabled {
		return nil
	}
	item.instance = nil
	item.metadata.Enabled = false
	return m.publish(ctx, id, item)
}
func (m *Manager) find(id string) (managedPlugin, error) {
	if m.closed {
		return managedPlugin{}, errors.New("plugin manager is closed")
	}
	item, ok := m.loaded[id]
	if !ok {
		return managedPlugin{}, fmt.Errorf("plugin %s is not installed", id)
	}
	return item, nil
}

// Upgrade preserves enabled state. New bytes are validated before changing
// durable state or the registry; in-flight snapshots retain the old version.
func (m *Manager) Upgrade(ctx context.Context, id string, archive []byte) (LoadedPlugin, error) {
	pkg, err := pluginpackage.Read(archive)
	if err != nil {
		return LoadedPlugin{}, err
	}
	if pkg.Manifest().ID != id {
		return LoadedPlugin{}, errors.New("upgrade plugin ID does not match")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	previous, err := m.find(id)
	if err != nil {
		return LoadedPlugin{}, err
	}
	item, err := m.prepare(ctx, pkg, archive, SourceInstalled, previous.metadata.Enabled)
	if err != nil {
		return LoadedPlugin{}, err
	}
	if err = m.publish(ctx, id, item); err != nil {
		return LoadedPlugin{}, err
	}
	return cloneLoaded(item.metadata), nil
}

// Uninstall removes installed bytes. For a bundled ID, retain only a disabled
// bundled record so restarting Lore cannot silently reactivate its embedded copy.
// Namespaced plugin data is intentionally retained for a possible reinstall.
func (m *Manager) Uninstall(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, err := m.find(id)
	if err != nil {
		return err
	}
	commit := func() error {
		if _, bundled := m.bundled[id]; bundled {
			return m.store.SavePlugin(ctx, Record{ID: id, Source: SourceBundled, Enabled: false})
		}
		return m.store.DeletePlugin(ctx, id)
	}
	var old *lifetime
	if item.instance != nil {
		old, err = m.registry.transition(id, nil, true, commit)
	} else {
		err = commit()
	}
	if err != nil {
		return err
	}
	delete(m.loaded, id)
	if archive, bundled := m.bundled[id]; bundled {
		pkg, err := pluginpackage.Read(archive)
		if err != nil {
			return err
		}
		m.loaded[id] = managedPlugin{archive: archive, metadata: LoadedPlugin{Manifest: pkg.Manifest(), Source: SourceBundled, Digest: pkg.Digest()}}
	}
	if _, bundled := m.bundled[id]; !bundled {
		m.order = slices.DeleteFunc(m.order, func(value string) bool { return value == id })
	}
	if item.instance != nil {
		m.retire(old, item.instance)
	}
	return nil
}

package plugin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"github.com/gi8lino/lore/internal/plugins/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRuntime groups the state and data associated with fake runtime.
type fakeRuntime struct {
	// instance owns the active executable plugin instance.
	instance *fakeInstance
	// err stores the terminal operation error.
	err error
	// closed indicates whether the associated resource is closed.
	closed bool
}

// Load loads the value.
func (r *fakeRuntime) Load(context.Context, *pluginpackage.Package) (plugin.Instance, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.instance, nil
}

// Close releases resources held by the receiver.
func (r *fakeRuntime) Close(context.Context) error { r.closed = true; return nil }

// fakeInstance groups the state and data associated with fake instance.
type fakeInstance struct {
	// closed indicates whether the associated resource is closed.
	closed bool
}

// Contributions returns the contributions owned by the instance.
func (i *fakeInstance) Contributions() plugin.Contributions { return plugin.Contributions{} }

// Close releases resources held by the receiver.
func (i *fakeInstance) Close(context.Context) error { i.closed = true; return nil }

// TestManagerRollsBackFailedRegistration verifies manager rolls back failed registration behavior.
func TestManagerRollsBackFailedRegistration(t *testing.T) {
	data, err := bundled.Packages.ReadFile("callouts.loreplugin")
	require.NoError(t, err)
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "io.lore.callouts", Name: "Existing"}, plugin.Contributions{}))
	runtime := &fakeRuntime{instance: &fakeInstance{}}
	manager := plugin.NewManager(registry, runtime)
	_, err = manager.Load(context.Background(), data, plugin.SourceInstalled)
	require.Error(t, err)
	assert.True(t, runtime.instance.closed)
	assert.Empty(t, manager.Plugins())
	assert.Equal(t, "Existing", registry.Snapshot().Entries[0].Descriptor.Name)
	require.NoError(t, manager.Close(context.Background()))
	assert.True(t, runtime.closed)
}

// TestManagerFailuresNeverPublishContributions verifies manager failures never publish contributions behavior.
func TestManagerFailuresNeverPublishContributions(t *testing.T) {
	data, err := bundled.Packages.ReadFile("callouts.loreplugin")
	require.NoError(t, err)
	registry := &plugin.Registry{}
	runtime := &fakeRuntime{err: errors.New("invalid reactor")}
	manager := plugin.NewManager(registry, runtime)
	_, err = manager.Load(context.Background(), data, plugin.SourceBundled)
	require.ErrorContains(t, err, "invalid reactor")
	assert.Empty(t, registry.Snapshot().Entries)
	_, err = manager.Load(context.Background(), []byte("invalid package"), plugin.SourceInstalled)
	require.Error(t, err)
	assert.Empty(t, registry.Snapshot().Entries)
	require.NoError(t, manager.Close(context.Background()))
	_, err = manager.Load(context.Background(), data, plugin.SourceBundled)
	require.ErrorContains(t, err, "closed")
}

// TestInstallCannotReplaceAnotherRegistryOwner verifies install cannot replace another registry owner behavior.
func TestInstallCannotReplaceAnotherRegistryOwner(t *testing.T) {
	data, err := bundled.Packages.ReadFile("callouts.loreplugin")
	require.NoError(t, err)
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "io.lore.callouts", Name: "Existing"}, plugin.Contributions{}))
	runtime := &fakeRuntime{instance: &fakeInstance{}}
	manager := plugin.NewManager(registry, runtime)
	_, err = manager.Install(context.Background(), data)
	require.ErrorContains(t, err, "already registered")
	assert.True(t, runtime.instance.closed)
	assert.Empty(t, manager.Plugins())
	assert.Equal(t, "Existing", registry.Snapshot().Entries[0].Descriptor.Name)
	require.NoError(t, manager.Close(context.Background()))
}

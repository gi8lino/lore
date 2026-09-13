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

type fakeRuntime struct {
	instance *fakeInstance
	err      error
	closed   bool
}

func (r *fakeRuntime) Load(context.Context, *pluginpackage.Package) (plugin.Instance, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.instance, nil
}
func (r *fakeRuntime) Close(context.Context) error { r.closed = true; return nil }

type fakeInstance struct{ closed bool }

func (i *fakeInstance) Contributions() plugin.Contributions { return plugin.Contributions{} }
func (i *fakeInstance) Close(context.Context) error         { i.closed = true; return nil }

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

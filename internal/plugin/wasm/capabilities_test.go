package wasm_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryStorage struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (s *memoryStorage) ReadPluginValue(_ context.Context, id, namespace, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[id+"/"+namespace+"/"+key]
	return bytes.Clone(value), ok, nil
}
func (s *memoryStorage) WritePluginValue(_ context.Context, id, namespace, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[id+"/"+namespace+"/"+key] = bytes.Clone(value)
	return nil
}
func capabilityPackage(t *testing.T, id string, permissions []string) *pluginpackage.Package {
	t.Helper()
	binary, err := fixtureWASM()
	require.NoError(t, err)
	grants, _ := json.Marshal(permissions)
	manifest := fmt.Sprintf("api_version: 1\nid: %s\nname: Fixture\nversion: 1.0.0\nmodules:\n  - type: renderer-extension\n    id: fixture\n    stage: preprocess\npermissions: %s\n", id, grants)
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for name, data := range map[string][]byte{"plugin.yaml": []byte(manifest), "plugin.wasm": binary} {
		file, err := archive.Create(name)
		require.NoError(t, err)
		_, err = file.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, archive.Close())
	pkg, err := pluginpackage.Read(output.Bytes())
	require.NoError(t, err)
	return pkg
}
func hostRequest(t *testing.T, instance plugin.Instance, ctx plugin.Context, method string, params any) string {
	t.Helper()
	data, err := json.Marshal(params)
	require.NoError(t, err)
	request, err := json.Marshal(pluginapi.CapabilityRequest{Method: method, Params: data})
	require.NoError(t, err)
	result, err := instance.Contributions().Preprocessors[0].Preprocess(ctx, "host:"+string(request))
	require.NoError(t, err)
	return result
}
func TestCapabilityPermissionsAndStorageIsolation(t *testing.T) {
	ctx := context.Background()
	permissions := []string{"pages:read", "storage:read", "storage:write", "settings:read", "settings:write"}
	storage := &memoryStorage{values: make(map[string][]byte)}
	runtime, err := wasm.New(ctx, wasm.Limits{}, wasm.WithPermissions(permissions...), wasm.WithStorage(storage))
	require.NoError(t, err)
	defer func() { _ = runtime.Close(ctx) }()
	instances := make([]plugin.Instance, 0, 2)
	for _, id := range []string{"io.one", "io.two"} {
		instance, err := runtime.Load(ctx, capabilityPackage(t, id, permissions))
		require.NoError(t, err)
		defer func() { _ = instance.Close(ctx) }()
		instances = append(instances, instance)
	}
	scope := plugin.Context{Context: ctx}
	for _, source := range []string{`host-raw:{"method":"plugin.storage.read","plugin_id":"io.two"}`, `host-raw:{"method":"log"} {}`} {
		got, err := instances[0].Contributions().Preprocessors[0].Preprocess(scope, source)
		require.NoError(t, err)
		assert.Contains(t, got, "invalid capability request")
	}
	got, err := instances[0].Contributions().Preprocessors[0].Preprocess(scope, "host-invalid-buffer")
	require.NoError(t, err)
	assert.Equal(t, "rejected", got)
	assert.Empty(t, storage.values)
	assert.Equal(t, "null", hostRequest(t, instances[0], scope, "plugin.storage.write", pluginapi.StorageValue{Key: "key", Value: []byte("one")}))
	assert.Contains(t, hostRequest(t, instances[0], scope, "plugin.storage.read", pluginapi.StorageValue{Key: "key"}), `"Found":true`)
	assert.Contains(t, hostRequest(t, instances[1], scope, "plugin.storage.read", pluginapi.StorageValue{Key: "key"}), `"Found":false`)
	assert.Contains(t, hostRequest(t, instances[0], scope, "plugin.settings.read", pluginapi.StorageValue{Key: "key"}), `"Found":false`)
	assert.Contains(t, hostRequest(t, instances[0], scope, "plugin.storage.read", map[string]any{"Key": "key", "plugin_id": "io.two"}), "invalid storage")
	assert.Contains(t, hostRequest(t, instances[0], scope, "network", nil), "denied")
	assert.Contains(t, hostRequest(t, instances[0], scope, "pages.get", nil), "unavailable")
	ungranted, err := wasm.New(ctx, wasm.Limits{})
	require.NoError(t, err)
	defer func() { _ = ungranted.Close(ctx) }()
	_, err = ungranted.Load(ctx, capabilityPackage(t, "io.denied", permissions))
	require.ErrorContains(t, err, "not granted")
	undeclared, err := runtime.Load(ctx, capabilityPackage(t, "io.undeclared", nil))
	require.NoError(t, err)
	defer func() { _ = undeclared.Close(ctx) }()
	assert.Contains(t, hostRequest(t, undeclared, scope, "plugin.storage.write", pluginapi.StorageValue{Key: "key"}), "denied")
	assert.Contains(t, hostRequest(t, undeclared, scope, "pages.get", nil), "denied")
}
func TestCapabilitiesUseCurrentRequestAndRecoverFromHostPanic(t *testing.T) {
	ctx := context.Background()
	runtime, err := wasm.New(ctx, wasm.Limits{}, wasm.WithPermissions("pages:read"))
	require.NoError(t, err)
	defer func() { _ = runtime.Close(ctx) }()
	instance, err := runtime.Load(ctx, capabilityPackage(t, "io.scope", []string{"pages:read"}))
	require.NoError(t, err)
	defer func() { _ = instance.Close(ctx) }()
	var workers sync.WaitGroup
	for _, viewer := range []string{"alice", "bob"} {
		workers.Go(func() {
			scope := plugin.Context{Context: ctx, Capabilities: map[string]plugin.Capability{"pages.get": func(context.Context, json.RawMessage) (any, error) { return viewer, nil }}}
			assert.Equal(t, `"`+viewer+`"`, hostRequest(t, instance, scope, "pages.get", nil))
		})
	}
	workers.Wait()
	scope := plugin.Context{Context: ctx, Capabilities: map[string]plugin.Capability{"pages.get": func(context.Context, json.RawMessage) (any, error) { panic("broken adapter") }}}
	assert.Contains(t, hostRequest(t, instance, scope, "pages.get", nil), "panicked")
	scope.Capabilities["pages.get"] = func(context.Context, json.RawMessage) (any, error) { return "healthy", nil }
	assert.Equal(t, `"healthy"`, hostRequest(t, instance, scope, "pages.get", nil))
}

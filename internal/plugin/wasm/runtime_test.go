package wasm_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"github.com/gi8lino/lore/internal/plugins/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtureWASM = sync.OnceValues(func() ([]byte, error) {
	temporary, err := os.MkdirTemp("", "lore-wasm-fixture-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	file := filepath.Join(temporary, "reactor.wasm")
	command := exec.Command("go", "build", "-buildmode=c-shared", "-ldflags=-s -w", "-o", file, "./testdata/reactor")
	command.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build test reactor: %w\n%s", err, output)
	}
	return os.ReadFile(file)
})

func archive(t *testing.T, wasmBytes []byte, stage string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range []struct {
		name string
		data []byte
	}{
		{"plugin.yaml", []byte("api_version: 1\nid: io.example.fixture\nname: Fixture\nversion: 1.0.0\nmodules:\n  - type: renderer-extension\n    id: fixture\n    stage: " + stage + "\npermissions: []\n")},
		{"plugin.wasm", wasmBytes},
	} {
		file, err := writer.Create(entry.name)
		require.NoError(t, err)
		_, err = file.Write(entry.data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func runtimeFixture(t *testing.T, stage string, limits wasm.Limits) (plugin.Instance, []byte) {
	t.Helper()
	compiled, err := fixtureWASM()
	require.NoError(t, err)
	data := archive(t, compiled, stage)
	pkg, err := pluginpackage.Read(data)
	require.NoError(t, err)
	runtime, err := wasm.New(context.Background(), limits)
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })
	instance, err := runtime.Load(context.Background(), pkg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = instance.Close(context.Background()) })
	return instance, data
}

func TestBundledAndInstalledCalloutsUseSameRuntime(t *testing.T) {
	data, err := bundled.Packages.ReadFile("callouts.loreplugin")
	require.NoError(t, err)
	var outputs []string
	for _, source := range []plugin.Source{plugin.SourceBundled, plugin.SourceInstalled} {
		runtime, err := wasm.New(context.Background(), wasm.Limits{})
		require.NoError(t, err)
		registry := &plugin.Registry{}
		manager := plugin.NewManager(registry, runtime)
		t.Cleanup(func() { _ = manager.Close(context.Background()) })
		metadata, err := manager.Load(context.Background(), data, source)
		require.NoError(t, err)
		assert.Equal(t, source, metadata.Source)
		assert.Equal(t, "1.0.0", metadata.Manifest.Version)
		renderer := markdown.NewWithRegistry(registry)
		got, err := renderer.Render("!!! warning\n!!! note\n**Nested** [[Page]]\n")
		require.NoError(t, err)
		outputs = append(outputs, got)
		assert.Contains(t, got, `class="callout warning"`)
		assert.Contains(t, got, `class="callout note"`)
		assert.Contains(t, got, "<strong>Nested</strong>")
		assert.Contains(t, got, `href="/pages/page"`)
		snapshot := registry.Snapshot()
		require.NoError(t, manager.Unload(context.Background(), metadata.Manifest.ID))
		require.Empty(t, registry.Snapshot().Entries)
		_, err = snapshot.Entries[0].Contributions.Preprocessors[0].Preprocess(plugin.Context{}, "text")
		require.ErrorContains(t, err, "closed")
		got, err = renderer.Render("!!! note\nDisabled\n")
		require.NoError(t, err)
		assert.NotContains(t, got, `class="callout`)
	}
	assert.Equal(t, outputs[0], outputs[1])
}

func TestSandboxDeniesAmbientCapabilities(t *testing.T) {
	instance, _ := runtimeFixture(t, "preprocess", wasm.Limits{})
	transform := instance.Contributions().Preprocessors[0]
	secret := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(secret, []byte("host secret"), 0600))
	t.Setenv("LORE_PLUGIN_TEST_SECRET", "host secret")
	environment, err := transform.Preprocess(plugin.Context{}, "environment")
	require.NoError(t, err)
	assert.Empty(t, environment)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	for _, source := range []string{"read:" + secret, "write:" + secret, "network:" + listener.Addr().String(), "process"} {
		got, err := transform.Preprocess(plugin.Context{}, source)
		require.NoError(t, err)
		assert.Equal(t, "denied", got, source)
	}
	remaining, err := os.ReadFile(secret)
	require.NoError(t, err)
	assert.Equal(t, "host secret", string(remaining))
}

func TestSandboxRejectsTrapsAndMalformedResultsAndRecovers(t *testing.T) {
	instance, _ := runtimeFixture(t, "preprocess", wasm.Limits{})
	transform := instance.Contributions().Preprocessors[0]
	for _, source := range []string{"trap", "bad-pointer", "oversized", "malformed", "trailing", "unknown-field", "grow"} {
		t.Run(source, func(t *testing.T) {
			_, err := transform.Preprocess(plugin.Context{}, source)
			require.Error(t, err)
			got, err := transform.Preprocess(plugin.Context{}, "healthy")
			require.NoError(t, err)
			assert.Equal(t, "healthy", got)
		})
	}
}

func TestSandboxTimeoutAndCancellation(t *testing.T) {
	instance, _ := runtimeFixture(t, "preprocess", wasm.Limits{CallTimeout: 100 * time.Millisecond})
	transform := instance.Contributions().Preprocessors[0]
	start := time.Now()
	_, err := transform.Preprocess(plugin.Context{}, "loop")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = transform.Preprocess(plugin.Context{Context: ctx}, "loop")
	require.Error(t, err)
	got, err := transform.Preprocess(plugin.Context{}, "healthy")
	require.NoError(t, err)
	assert.Equal(t, "healthy", got)
}

func TestSandboxMemoryLimitAppliesAtInstantiation(t *testing.T) {
	compiled, err := fixtureWASM()
	require.NoError(t, err)
	pkg, err := pluginpackage.Read(archive(t, compiled, "preprocess"))
	require.NoError(t, err)
	runtime, err := wasm.New(context.Background(), wasm.Limits{MemoryPages: 1})
	require.NoError(t, err)
	defer func() { _ = runtime.Close(context.Background()) }()
	_, err = runtime.Load(context.Background(), pkg)
	require.Error(t, err)
}

func TestSandboxWireLimitAndPostprocessorContract(t *testing.T) {
	instance, _ := runtimeFixture(t, "postprocess", wasm.Limits{WireBytes: 1024})
	transform := instance.Contributions().Postprocessors[0]
	_, err := transform.Postprocess(plugin.Context{}, strings.Repeat("x", 1025))
	require.ErrorContains(t, err, "size limit")
	_, err = transform.Postprocess(plugin.Context{}, "recursive")
	require.ErrorContains(t, err, "fragment")
	got, err := transform.Postprocess(plugin.Context{}, "healthy")
	require.NoError(t, err)
	assert.Equal(t, "healthy", got)
}

func TestWASMOutputCannotBypassSanitizer(t *testing.T) {
	instance, _ := runtimeFixture(t, "preprocess", wasm.Limits{})
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "fixture", Name: "Fixture"}, instance.Contributions()))
	renderer := markdown.NewWithRegistry(registry)
	got, err := renderer.Render("unsafe")
	require.NoError(t, err)
	assert.Contains(t, got, "safe")
	assert.NotContains(t, got, "<script")
	assert.NotContains(t, got, "javascript:")
	_, err = renderer.Render("recursive")
	require.ErrorContains(t, err, "nesting limit")
}

func TestWASMRequestsAreIsolatedAndSerialized(t *testing.T) {
	instance, _ := runtimeFixture(t, "preprocess", wasm.Limits{})
	transform := instance.Contributions().Preprocessors[0]
	var wg sync.WaitGroup
	for index := range 12 {
		wg.Go(func() {
			source := fmt.Sprintf("request-%d", index)
			got, err := transform.Preprocess(plugin.Context{}, source)
			assert.NoError(t, err)
			assert.Equal(t, source, got)
		})
	}
	wg.Wait()
}

func TestRuntimeRejectsMissingABI(t *testing.T) {
	pkg, err := pluginpackage.Read(archive(t, []byte{0, 'a', 's', 'm', 1, 0, 0, 0}, "preprocess"))
	require.NoError(t, err)
	runtime, err := wasm.New(context.Background(), wasm.Limits{})
	require.NoError(t, err)
	defer func() { _ = runtime.Close(context.Background()) }()
	_, err = runtime.Load(context.Background(), pkg)
	require.ErrorContains(t, err, "memory")
}

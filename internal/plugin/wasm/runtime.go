// Package wasm adapts untrusted WASI reactors to Lore's module contracts.
package wasm

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/pluginpackage"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Limits bounds individual sandbox calls, guest memory, and wire payloads.
// Each page is 64 KiB. Zero fields use conservative defaults.
type Limits struct {
	MemoryPages uint32
	CallTimeout time.Duration
	LoadTimeout time.Duration
	WireBytes   int
	Parts       int
}

func (l Limits) defaults() Limits {
	if l.MemoryPages == 0 {
		l.MemoryPages = 1024
	}
	if l.CallTimeout <= 0 {
		l.CallTimeout = 2 * time.Second
	}
	if l.LoadTimeout <= 0 {
		l.LoadTimeout = 60 * time.Second
	}
	if l.WireBytes <= 0 {
		l.WireBytes = 4 << 20
	}
	if l.Parts <= 0 {
		l.Parts = 256
	}
	return l
}

// The compilation cache shares machine code, never registries, guest memory,
// request state, or capabilities. It is in memory only and lives for the process.
var compilationCache = wazero.NewCompilationCache()

// Wazero's cache does not single-flight concurrent compilations. Serialize
// loading to avoid compiling the same binary many times during parallel startup.
// This gate never covers rendering or shares guest state.
var compilationGate = make(chan struct{}, 1)

type Runtime struct {
	engine wazero.Runtime
	limits Limits
}

func New(ctx context.Context, limits Limits) (*Runtime, error) {
	limits = limits.defaults()
	if limits.MemoryPages > 65536 || limits.WireBytes > 16<<20 || limits.Parts > 4096 {
		return nil, errors.New("invalid WASM runtime limits")
	}
	config := wazero.NewRuntimeConfig().WithMemoryLimitPages(limits.MemoryPages).
		WithCloseOnContextDone(true).WithCompilationCache(compilationCache)
	engine := wazero.NewRuntimeWithConfig(ctx, config)
	// No filesystem preopens, environment, process arguments, sockets, or host
	// streams are configured. WASI descriptors cannot access Lore's resources.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, engine); err != nil {
		_ = engine.Close(ctx)
		return nil, err
	}
	return &Runtime{engine: engine, limits: limits}, nil
}

func (r *Runtime) Load(ctx context.Context, pkg *pluginpackage.Package) (plugin.Instance, error) {
	ctx, cancel := context.WithTimeout(ctx, r.limits.LoadTimeout)
	defer cancel()
	select {
	case compilationGate <- struct{}{}:
		defer func() { <-compilationGate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	binary := pkg.WASM()
	compiled, err := r.engine.CompileModule(ctx, binary)
	if err != nil {
		return nil, fmt.Errorf("compile WASM: %w", err)
	}
	if err := validateABI(compiled); err != nil {
		_ = compiled.Close(ctx)
		return nil, err
	}
	instance := &Instance{runtime: r, compiled: compiled, manifest: pkg.Manifest(), gate: make(chan struct{}, 1)}
	initializeCtx, stop := context.WithTimeout(ctx, r.limits.CallTimeout)
	defer stop()
	if err := instance.instantiate(initializeCtx); err != nil {
		_ = compiled.Close(ctx)
		return nil, err
	}
	if err := r.retainCode(ctx, binary); err != nil {
		_ = instance.Close(context.Background())
		return nil, err
	}
	return instance, nil
}

func (r *Runtime) Close(ctx context.Context) error { return r.engine.Close(ctx) }

func validateABI(compiled wazero.CompiledModule) error {
	if len(compiled.ImportedMemories()) != 0 {
		return errors.New("imported WASM memory is not allowed")
	}
	for _, imported := range compiled.ImportedFunctions() {
		namespace, name, _ := imported.Import()
		if namespace != wasi_snapshot_preview1.ModuleName {
			return fmt.Errorf("unsupported WASM import %s.%s", namespace, name)
		}
	}
	if compiled.ExportedMemories()["memory"] == nil {
		return errors.New("plugin must export memory")
	}
	signatures := []struct {
		name            string
		params, results []api.ValueType
	}{
		{"_initialize", nil, nil},
		{"lore_api_version", nil, []api.ValueType{api.ValueTypeI32}},
		{"lore_alloc", []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}},
		{"lore_transform", []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}},
	}
	for _, signature := range signatures {
		function := compiled.ExportedFunctions()[signature.name]
		if function == nil || !slices.Equal(function.ParamTypes(), signature.params) || !slices.Equal(function.ResultTypes(), signature.results) {
			return fmt.Errorf("missing or incompatible WASM export %s", signature.name)
		}
	}
	return nil
}

func (i *Instance) instantiate(ctx context.Context) error {
	// Anonymous instances cannot be imported by another plugin. Only _initialize
	// is invoked, and it shares the load/call deadline and memory limit.
	module, err := i.runtime.engine.InstantiateModule(ctx, i.compiled, wazero.NewModuleConfig().WithName("").WithStartFunctions("_initialize"))
	if err != nil {
		return fmt.Errorf("initialize WASM: %w", err)
	}
	version, err := module.ExportedFunction("lore_api_version").Call(ctx)
	if err != nil || len(version) != 1 || version[0] != pluginapi.Version {
		_ = module.Close(context.Background())
		return errors.New("incompatible WASM plugin API version")
	}
	i.module = module
	return nil
}

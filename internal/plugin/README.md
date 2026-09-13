# Plugin architecture (Phases 1–2)

Lore's rendering modules register contributions through an application-owned
`plugin.Registry`. `markdown.New(ctx)` creates a renderer with an owned manager
and WASM runtime; callers handle startup errors and close the renderer at the end
of its scope. `markdown.NewWithRegistry` supports an explicitly owned registry.
Server pages, preview, sharing, exports, and static builds use the same pipeline.

Callouts is now a real bundled `.loreplugin`, compiled from the independent Go
module in `plugins/callouts`. Its native implementation has been removed.
Subpages and Page Report still use native adapters pending Phase 3.

## Packages and distribution

A `.loreplugin` is a ZIP containing:

```text
plugin.yaml
plugin.wasm
assets/          # optional; not served or executed yet
```

The versioned manifest declares identity, a numeric `major.minor.patch` version,
modules, dependencies, presentation defaults, and permissions. Phase 2 supports
`renderer-extension` modules at `preprocess` and `postprocess` stages. Unknown
fields, unsupported API versions, stages, module types, and nonempty permission
requests are rejected. No capability is silently granted.

`pluginpackage.Read` validates the whole archive before returning immutable
content. It rejects traversal, absolute paths, backslashes, duplicate paths,
nonregular files, invalid directories, and file/directory collisions. It never
extracts into the filesystem. Limits are 16 MiB compressed, 32 MiB expanded,
256 entries, 16 MiB WASM, 4 MiB per asset, and 64 KiB for the manifest. CRC and
actual decompressed-size checks apply when entries are read.

`internal/plugins/bundled` embeds the package bytes. Bundled loading calls the
same `Manager.Load` method as a manually supplied package. `SourceBundled` and
`SourceInstalled` are descriptive metadata only: they do not change validation,
runtime configuration, permissions, or rendering behavior. There is no separate
native or privileged path for bundled Callouts.

## Registration and lifetime

`Register(Descriptor, Contributions)` publishes a complete contribution set
atomically. Duplicate IDs and macro names, invalid metadata, and unavailable
dependencies fail without publishing partial contributions. Requirements load
first and unload after dependents.

Each top-level render takes one snapshot. Nested blocks and variable-provenance
passes keep that view; registry locks are not held during rendering. Native
callbacks must be immutable and concurrency safe. Request macro bindings are
copied and cannot reactivate an unregistered macro.

`Manager.Load` validates, compiles, instantiates, then registers. It closes the
new instance if registration fails. `Unload` unregisters and closes it. Current
WASM calls drain; a stale snapshot that has not entered a closed instance gets a
render error. `Close` releases runtime resources when the server or build ends.

These are in-memory loading primitives, not a persistent installer or complete
runtime lifecycle. Phase 4 must add persisted state, install/upgrade/uninstall,
and instance leases for uninterrupted in-flight snapshots during replacement.
The manager deliberately has no database/schema or plugin-directory dependency
in Phase 2.

## Runtime boundary

`internal/plugin/wasm` hosts WASI reactors with wazero. Each instance has its own
linear memory and serialized invocation gate. No host filesystem, environment,
arguments, sockets, or streams are configured. Only WASI imports are accepted;
imported memories and cross-plugin module imports are rejected. API export
signatures and the guest's API version are checked before registration.

Defaults are 64 MiB guest memory, a 2-second call deadline, a 60-second compilation/load
deadline and a separate 2-second initialization deadline, 4 MiB request/response and assembled-output limits, and 256 output
fragments. Limits are configurable at runtime construction. The renderer also
limits recursive depth to 64 and a complete render to 30 seconds, respecting an
earlier request cancellation. Compilation has archive-size and elapsed-time
bounds, but not a separate hard cap on the compiler's host-memory usage.

Traps, deadline exhaustion, bad offsets, incompatible responses, and guest errors
become render errors. The bad reactor is closed; a later request can instantiate
a clean reactor from the compiled module. Nested Markdown fragments are handled
after the guest invocation, so there is no reentrant call into a Go WASM runtime.

A process-local, memory-only compilation cache shares code, never active
registries, guest state, or permissions. Up to eight compiled-code references
are retained for reuse across short-lived renderers and evicted in least-recently-used order.
A startup gate prevents duplicate
concurrent compilations, following
[wazero's compilation-cache guidance](https://pkg.go.dev/github.com/tetratelabs/wazero#CompilationCache).

The wire ABI is documented in `pluginapi/README.md`. Goldmark extension objects
remain host-side adapters; WASM guests never receive Goldmark or Lore pointers.
Later AST support must use serialized operations available to all plugins.

## Rendering and sanitization

The pipeline recognizes registered macros outside CommonMark code, runs Markdown
preprocessors and core features, constructs fresh Goldmark extensions, expands
macros, then runs HTML postprocessors and the central sanitizer. Existing
Tabs/Details ordering and macro-heading table-of-contents behavior are preserved.
Browser, editor, and settings registry contributions currently hold metadata only.

Callouts' existing administrator preference is translated at the composition
boundary. The runtime adapter honors a configured plugin feature flag generically;
there is no switch on Callouts' ID in the runtime. Its existing CSS remains in
Lore's frontend until browser assets are migrated.

Every output path, including WASM fragments and macros, goes through the core
sanitizer. Plugins cannot mark HTML trusted or change the policy. A restricted,
static SVG geometry allowlist preserves navigation icons while rejecting active
SVG, URL-valued paints, scripts, and event handlers.

## Building and validation

`make plugin-packages` builds standard-Go WASI reactors and deterministic ZIP
archives. The guest's `go.mod` pins its compiler version; trimpath, disabled VCS
metadata, an empty build ID, sorted files, and fixed ZIP metadata make artifacts
reproducible. Packages are committed and embedded, so ordinary Lore compilation
and Docker builds need no guest compiler invocation.

`make generate` regenerates packages and icons. `make check-generated` compares
the checked-in packages. `make test` and `make test-race` also test the standalone
plugin sources. Runtime tests execute both real bundled Callouts and an adversarial
WASM fixture, including ambient-capability denial, traps, malformed output,
resource limits, request isolation, and sanitizer enforcement.

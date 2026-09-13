# Plugin architecture (Phases 1–4)

Lore's rendering modules register contributions through an application-owned
`plugin.Registry`. `markdown.New(ctx)` creates a renderer with an owned manager
and WASM runtime; `NewWithPluginStore` also restores persisted lifecycle state; callers handle startup errors and close the renderer at the end
of its scope. `markdown.NewWithRegistry` supports an explicitly owned registry.
Server pages, preview, sharing, exports, and static builds use the same pipeline.

Callouts is now a real bundled `.loreplugin`, compiled from the independent Go
module in `plugins/callouts`. Its native implementation has been removed.
Subpages and Page Report are also bundled WASM macro plugins. Their feature
implementations live under `plugins/features` and import only public wire types;
the old native implementations have been removed.

## Packages and distribution

A `.loreplugin` is a ZIP containing:

```text
plugin.yaml
plugin.wasm
assets/          # optional; not served or executed yet
```

The versioned manifest declares identity, a numeric `major.minor.patch` version,
modules, dependencies, presentation defaults, and permissions. The manifest supports
`renderer-extension` modules at `preprocess` and `postprocess` stages and `macro`
modules with parse/render calls. Unknown fields, unsupported versions, stages,
module types, and permission names are rejected. No capability is silently granted.

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

Each top-level render acquires and releases one leased snapshot. Nested blocks and variable-provenance
passes keep that view; registry locks are not held during rendering. Native
callbacks must be immutable and concurrency safe. Request macro bindings are
copied and cannot reactivate an unregistered macro.

`Manager.Install`, `Enable`, `Disable`, `Upgrade`, and `Uninstall` work inside
one running process. The candidate package is validated and instantiated before
publication. Registry validation, durable state commit, and publication are
ordered so validation or persistence failures leave the active version intact.
Replacement preserves contribution order and checks the complete dependency
graph, including cycles introduced by upgrades. Required plugins cannot be
disabled or removed while an enabled dependent still needs them.

A render leases every contribution version in its snapshot. Removing or replacing
an entry affects new renders immediately; the old WASM instance closes after its
last render releases the lease. Registry locks never cover guest execution.
Shutdown detaches all owned contributions atomically and waits for retired
instances. A cancelled shutdown can be retried. `Snapshot()` is an unleased
inspection API; executable consumers use `Acquire()` and release on every path.

The manager's small `Store` interface persists installation records. Server
composition supplies PostgreSQL through `NewWithPluginStore`. The
`plugin_installations` table stores installed ZIP bytes separately from embedded
bundled packages, alongside source and enabled state. This uses Lore's existing
persistence abstraction and requires no fixed filesystem path. The default
in-memory store supports isolated renderers and standalone static builds.

Startup merges bundled packages with persisted overrides, orders enabled plugins
by dependency, prepares every instance, and publishes the complete registry once.
A failed startup closes all prepared instances and publishes nothing. Bundled
state records never contain package bytes. Upgrading a bundled ID creates an
installed override through the same loader/runtime. Uninstall removes installed
bytes; if that ID has an embedded copy, the embedded copy remains disabled so a
restart cannot silently reactivate it. Plugin settings/data are retained for
reinstallation. Version replacement requires the same ID and preserves enabled
state; explicit replacements may also restore an earlier version.

`Load`/`Unload` remain transient, low-level helpers for explicitly owned managers;
application installation uses the durable lifecycle methods. The application
exposes its manager through `Renderer.PluginManager()`. Administration routes and
UI remain Phase 7. The standalone site CLI uses bundled defaults;
`site.BuildWithRenderer` lets an application reuse its live registry for builds.

## Runtime boundary

`internal/plugin/wasm` hosts WASI reactors with wazero. Each instance has its own
linear memory and serialized invocation gate. No host filesystem, environment,
arguments, sockets, or streams are configured. Only WASI and the signature-checked `lore_v1.call` import are accepted;
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

Process-local, bounded caches share validated package values and immutable
compiled modules, never active registries, guest state, or permissions. Eight
entries are retained in each least-recently-used cache. Compiled-module leases
keep an evicted entry alive until its active instances close. This avoids both
repeated ZIP expansion and repeated WASM decoding across renderer scopes.
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

## Capability and storage boundary

`pluginapi` defines public JSON values and a Go guest transport. It never imports
`internal/domain`, `internal/store`, or `internal/handler`. `plugincap` is the
trusted composition adapter: it converts already-authorized catalogs and
navigation into public values. Normal pages, previews and exports keep the
viewer's existing access filter. Anonymous share scopes expose only the shared
page. Static rendering exposes prepared navigation with static URLs and leaves
unavailable query macros literal.

Runtime policy and manifest declarations must both allow each sensitive host
call. The caller identity is derived from the executing WASM instance and is
never accepted in request JSON. Capabilities are bound to the current invocation,
not stored in the instance; concurrent viewers cannot share callbacks. Calls
outside an invocation, unknown operations, malformed buffers, and undeclared
permissions are denied. A host-adapter panic becomes a guest-visible error.

The application explicitly grants page reads and namespaced settings/data
permissions to requesting packages. The lower-level runtime grants nothing by
default. Attachments have an optional authorized range-reader adapter and are
not bound automatically. No network, user directory, general settings, SQL,
filesystem, or process capability is available. The same grant rules apply to
both distribution sources.

Storage is keyed by plugin ID, namespace (`settings` or `data`), and key. IDs
come from the runtime. Limits are 256-byte keys, 64 KiB per value, 1,024 keys and
16 MiB total per plugin. A PostgreSQL transaction and per-plugin advisory lock
make quota checks atomic. This storage survives runtime restarts; plugin installation state is stored separately, and administration remains a later phase.

See `pluginapi/README.md` for methods and wire contracts. Tests cover actual WASM
macro parity between bundled and installed packages, authorization failures,
request isolation, host panics, malformed requests, namespace forgery, quota
checks, and PostgreSQL reopen persistence. Rebuild packages with
`make plugin-packages` after changing either wire types or plugin source.

## Lifecycle validation

Tests exercise install/disable/re-enable/upgrade/uninstall through real WASM in
one process, a render spanning an upgrade, dependency and cycle failures,
persistence failure rollback, bootstrap atomicity, installed overrides of bundled
IDs, and PostgreSQL/runtime reopen recovery. A static build test reuses the live
renderer across disable/re-enable. No installation UI, asset server, marketplace,
or remote package downloader is introduced in this phase.

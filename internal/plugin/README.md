# Plugin foundation (Phase 1)

Lore's rendering modules register contributions through an application-owned
`plugin.Registry`. `markdown.DefaultRegistry` composes the currently bundled
modules, while `markdown.NewWithRegistry` accepts an explicit registry. The server
passes its renderer to page, preview, sharing, and export handlers. Static builds
use the same default composition and rendering pipeline.

This is an internal foundation, not yet a supported community SDK or an installer.
Only compiled, trusted modules execute natively. There is no Go `plugin` loading,
WASM runtime, package format, plugin persistence, or browser asset loader yet.

## Registration and lifetime

`Register(Descriptor, Contributions)` publishes one owner's complete contribution
set atomically. Duplicate plugin IDs, duplicate macro names, invalid metadata,
and missing dependencies fail without publishing partial contributions. Load
requirements first; unregister dependents before their requirements.

`Unregister(id)` removes all of that plugin's contributions. Each top-level render
takes one snapshot; nested blocks and variable-provenance passes keep that view.
Already-started renders can finish with old contributions, while subsequent
renders see the new registry. Callbacks must be immutable and concurrency safe.
Registry locks are never held while rendering.

A later runtime manager must lease instances until their in-flight snapshots
finish before closing them. The registry alone does not manage runtime instances,
persist enabled state, or implement safe version replacement. `DefaultEnabled`
is metadata used by composition, not a second mutable activation flag.

## Rendering contracts

The pipeline is:

1. Recognize registered standalone macros outside CommonMark code blocks and
   preserve their positions with per-render, unpredictable placeholders.
2. Run Markdown preprocessors and the remaining core features. Nested block
   rendering uses the same contribution snapshot.
3. Construct fresh Goldmark extensions, including their AST transformers and
   node renderers, and convert Markdown into intermediate HTML.
4. Expand registered macros with request-local bindings.
5. Run HTML postprocessors once over the complete document, including macros.
6. Apply the trusted core sanitizer and return HTML.

Existing tabs/details transformations remain before the module preprocessors to
preserve their nesting behavior. Other optional core features have deliberately
not been migrated in this phase. Generated macro headings remain outside the
source page's table of contents, preserving existing behavior.

`Preprocessor`, `MarkdownExtension`, `Postprocessor`, and `Macro` are separate
interfaces. Browser, editor, and settings contributions currently hold metadata
only; registering them does not load code, serve files, or grant capabilities.

A preprocessor receives only render-local feature flags, macro bindings, and a
Markdown-rendering callback. Macro arguments are JSON. `BoundMacro` and
`BindMacro` adapt the existing typed Subpages and Page Report parsers/renderers;
request bindings provide their existing authorized navigation/query contexts.
Bindings cannot reactivate a macro removed from the registry. An unbound Subpages
macro stays empty; an unbound Page Report stays literal, as before.

Callouts lives in `internal/plugins/callouts`. It receives no renderer instance,
sanitizer, database, HTTP handler, or privileged bundled-only API. Its legacy
setting is translated to a feature flag at the composition boundary. Registering
or removing Callouts affects an existing renderer's next call.

## Trust boundary and later WASM adapter

Every macro and module returns untrusted intermediate HTML. No contribution can
alter the core sanitizer or mark its output trusted. The sanitizer permits the
static navigation and report markup previously inserted after sanitization,
including a restricted geometry-only SVG subset for existing icons. Active SVG,
URL-valued SVG paints, event handlers, scripts, and unsafe links are rejected.

Synchronous module panics become rendering errors and recursive Markdown calls
have a depth bound. These measures are not a sandbox: native goroutines, CPU,
memory, and I/O remain trusted until the WASM runtime is introduced.

The Goldmark interface is explicitly a host-side adapter, not a future WASM ABI.
A runtime adapter can implement the same stage interfaces using serialized
operations. Community binaries must never receive Go pointers or load native
callbacks. `Context.RenderMarkdown` will require a bounded host call; the existing
request bindings will be replaced by permission-checked capabilities in Phase 3.
Bundled and installed packages must use that same adapter and capability path.

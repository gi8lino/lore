Refactor Lore toward a Confluence-style runtime plugin system in multiple phases.

The final architecture should allow bundled first-party plugins and user-installed community plugins to use the same plugin model.

User-installed plugins should eventually be installable, enableable, disableable, upgradeable, and removable without recompiling or restarting Lore.

Do not attempt the entire plugin system in one change.

The phases below should be implemented separately, with Lore building and tests passing after each phase.

# Final architecture

The intended end state is:

```text
Lore Core
├── authentication / authorization
├── page primitives
├── persistence primitives
├── HTTP infrastructure
├── rendering pipeline
├── central sanitizer
├── plugin manager
├── plugin registry
├── plugin runtime
├── plugin permissions
├── plugin storage
└── public Plugin API
          │
          ▼
     Plugin Runtime
          │
     ┌────┴────┐
     │         │
 bundled     installed
 plugins      plugins
     │         │
     └── WASM ─┘
```

Bundled plugins must not receive privileged APIs unavailable to community plugins.

“Bundled” should describe distribution only.

A bundled plugin and the same plugin installed manually should use the same:

```text
manifest
plugin API
runtime
registry
permissions
storage
render pipeline
asset handling
```

Do not use Go's standard-library `plugin` package.

The target runtime for dynamically installed plugins is WASM/WASI, preferably hosted with wazero.

Do not implement all of that in Phase 1.

---

# Phase 1 — Plugin architecture foundation

Goal: introduce the plugin/module architecture without introducing WASM yet.

This phase should fit into one focused Codex run.

## Introduce a plugin registry

Create a registry that owns active plugin/module contributions.

Avoid a global mutable registry where practical.

Conceptually:

```text
Registry
├── renderer extensions
├── macros
├── browser modules
├── editor extensions
└── settings modules
```

Use small interfaces rather than one giant plugin interface.

A possible starting point:

```go
type Plugin interface {
    Descriptor() Descriptor
    Register(*Registry) error
}

type Descriptor struct {
    ID             string
    Name           string
    Description    string
    DefaultEnabled bool
    Requires       []string
}
```

The exact interface may differ if a cleaner Go design emerges.

The architecture must allow registrations to be removed later, because runtime disable/uninstall will be added in a later phase.

## Refactor the rendering pipeline

The current renderer contains too much feature-specific knowledge.

Move toward:

```text
Markdown
   ↓
preprocessors
   ↓
Goldmark extensions
   ↓
AST transformers / node renderers
   ↓
HTML postprocessors
   ↓
central sanitizer
   ↓
HTML
```

Plugins/modules should be able to contribute to these stages.

Keep sanitization inside trusted Lore core.

Plugin output must not bypass it.

## Generalize page functions/macros

Remove the central renderer's hardcoded knowledge of:

```text
Subpages
PageReport
```

Introduce a generic macro/page-function registry.

Conceptually:

```go
type Macro interface {
    Name() string
    Parse(string) (Invocation, bool)
    Render(Context, Invocation) (string, error)
}
```

Use another shape if it produces a cleaner API.

The central renderer should not switch on specific macro names.

The architecture must support future macros such as:

```text
{{subpages}}
{{pages ...}}
{{toc}}
{{status ...}}
{{include ...}}
```

Macro syntax inside fenced code must remain literal.

## Convert one simple feature

Convert Callouts to use the new internal plugin/module mechanism.

Callouts should become the proof that the renderer no longer needs hardcoded feature handling.

Do not implement WASM yet.

Callouts may still execute natively in this phase, but they must use the same abstractions intended for runtime plugins later.

## Constraints

Preserve existing behavior.

Do not:

```text
add WASM yet
add plugin installation UI
add marketplace support
migrate every rendering feature
change persistence unnecessarily
```

Keep all tests passing.

The most important outcome of Phase 1 is a clean API that later phases can put behind a WASM boundary.

Suggested commit:

```text
refactor: introduce plugin rendering architecture
```

---

# Phase 2 — Plugin package and WASM runtime

Goal: make the plugin architecture executable through WASM.

## Add a `.loreplugin` package format

A plugin package should conceptually contain:

```text
plugin.yaml
plugin.wasm
assets/
    plugin.js
    plugin.css
```

Use zip or tar depending on what fits Lore best.

Version the manifest format from the beginning.

Example:

```yaml
api_version: 1

id: io.lore.callouts
name: Callouts
version: 1.0.0

modules:
  - type: renderer-extension
    id: callouts

permissions: []
```

Keep the first manifest small.

## Add a WASM runtime

Prefer wazero unless there is a strong technical reason otherwise.

Lore should be able to:

```text
read manifest
validate plugin
check compatibility
instantiate plugin.wasm
register modules
stop the instance
```

Do not give plugins unrestricted access to:

```text
filesystem
environment
database
network
process execution
```

## Add a plugin manager

Create a real plugin-manager/domain layer.

Conceptually:

```go
type Manager struct {
    registry Registry
    runtime  Runtime
    store    Store
}
```

Responsibilities should eventually include:

```text
discover
load
enable
disable
install
upgrade
uninstall
```

Only implement what this phase needs.

## Bundle Callouts as an actual plugin

Convert the Phase 1 Callouts implementation into a real bundled `.loreplugin`.

Embed it into the Lore binary.

Conceptually:

```go
//go:embed plugins/*.loreplugin
var bundledPlugins embed.FS
```

Bundled plugins must go through the same package loader and runtime that installed plugins will later use.

Suggested commit:

```text
feat: add wasm plugin runtime
```

---

# Phase 3 — Plugin API, permissions, and storage

Goal: provide plugins with stable capabilities without exposing Lore internals.

## Public capability API

Do not expose:

```text
internal/domain
internal/store
internal/handler
```

to plugins.

Expose capabilities such as:

```text
Pages
Search
Attachments
Settings
PluginStorage
Logging
```

Conceptually:

```text
pages.get
pages.search
attachments.read
plugin.settings.read
plugin.settings.write
plugin.storage.read
plugin.storage.write
log
```

Bundled plugins must use these same APIs.

Rule:

> If a bundled plugin needs functionality unavailable through the public Plugin API, improve the Plugin API instead of giving the bundled plugin privileged access.

## Permissions

Introduce a permission model.

Possible permissions:

```text
pages:read
pages:write
users:read
attachments:read
network
```

Host calls must be permission checked.

Plugins should receive no sensitive capability implicitly.

## Plugin storage

Do not let plugins modify Lore's core schema.

Provide namespaced storage.

Conceptually:

```text
plugin_settings
---------------
plugin_id
key
value

plugin_data
-----------
plugin_id
key
value
```

The exact schema can differ.

Plugins access this storage only through the Plugin API.

## Convert Subpages and Page Report

Use them to validate the capability API.

Subpages validates page/context access.

Page Report validates search/query access.

Neither should directly access Lore internals once migrated.

Suggested commit:

```text
feat: add plugin capabilities and storage
```

---

# Phase 4 — Runtime install/enable/disable/upgrade/uninstall

Goal: make user-installed plugins work without rebuilding or restarting Lore.

Implement runtime lifecycle:

```text
Install
Enable
Disable
Upgrade
Uninstall
```

Install should conceptually perform:

```text
plugin package
    ↓
validate
    ↓
persist
    ↓
instantiate WASM
    ↓
register modules
    ↓
available immediately
```

Disable:

```text
plugin
   ↓
unregister modules
   ↓
close WASM instance
   ↓
disabled
```

No Lore restart should be necessary.

Registrations must therefore be reversible.

Upgrade should safely replace the active plugin version.

Installed plugins should be stored separately from bundled plugins.

For example:

```text
/data/plugins/
```

Do not hardcode that exact location if Lore already has a better persistent-data abstraction.

Add tests proving install/disable/re-enable works within one running Lore process.

Suggested commit:

```text
feat: support runtime plugin lifecycle
```

---

# Phase 5 — Dynamic browser assets and Mermaid

Goal: validate plugins that contain frontend behavior.

Plugins may contribute:

```text
JavaScript
CSS
```

These assets must not require rebuilding Lore's main frontend bundle.

Expose plugin assets dynamically, for example:

```text
/plugins/{plugin-id}/assets/plugin.js
/plugins/{plugin-id}/assets/plugin.css
```

Only enabled plugins should have their browser assets loaded.

Prevent traversal outside the plugin package.

## Convert Mermaid

Move Mermaid behind the plugin system.

Mermaid should validate:

```text
browser assets
plugin enable/disable
render integration
admin metadata
preview behavior
```

A disabled Mermaid plugin should stop loading its browser module without requiring a Lore restart.

Suggested commit:

```text
feat: support plugin browser modules
```

---

# Phase 6 — Tables as the architecture stress test

Goal: prove the plugin model supports a complex first-party feature.

Convert Tables to a real plugin.

Tables currently span multiple concerns:

```text
Goldmark table extension
{table ...} directives
HTML postprocessing
table styling
sorting
filtering
TypeScript
CSS
admin settings
dependency rules
sanitizer requirements
```

The plugin should own these concerns as much as possible.

The central renderer must not contain table-specific logic after migration.

Dependencies should become declarative, for example:

```text
table-styles     -> tables
table-sorting    -> tables
table-filtering  -> tables
```

Do not keep special-case table dependency validation inside HTTP handlers if the plugin/module registry can own it.

Important rule:

> If Tables cannot be implemented cleanly through the same API available to community plugins, improve the plugin API rather than adding a Tables-specific escape hatch.

Suggested commit:

```text
refactor: move tables into plugin system
```

---

# Phase 7 — Plugin administration UI

Goal: expose plugin lifecycle through Lore's admin interface.

Add:

```text
Administration
└── Plugins
```

Conceptually:

```text
Callouts      Lore       Bundled       Enabled
Tables        Lore       Bundled       Enabled
Mermaid       Lore       Bundled       Enabled
GitHub        Community  Installed     Enabled
PlantUML      Community  Installed     Disabled
```

Plugin detail pages should eventually show:

```text
name
version
provider
source
status
modules
permissions
dependencies
settings
```

Allow:

```text
upload/install
enable
disable
upgrade
uninstall
```

Do not implement a public marketplace yet.

Required/system plugins can be marked as non-disableable.

Suggested commit:

```text
feat: add plugin administration
```

---

# Phase 8 — Migrate remaining optional Lore features

Once the API has been proven by the earlier plugins, migrate additional optional functionality.

Candidates:

```text
wiki links
tabs
details
task lists
syntax highlighting
footnotes
definition lists
strikethrough
autolinks
typographer
```

Do not migrate basic CommonMark unnecessarily.

Keep foundational Markdown/core rendering native where that is simpler and more appropriate.

General rendering preferences such as:

```text
ContentLanguage
DefaultTypographySize
```

should remain ordinary Lore settings rather than becoming plugins.

Suggested commits should be feature-specific rather than one giant migration.

---

# Phase 9 — Go Plugin SDK and developer tooling

Goal: make community plugin development pleasant.

Plugin authors should write ordinary Go, not raw WASM ABI code.

Provide or create a separate Lore Plugin SDK.

Developer experience should eventually be:

```sh
lore-plugin init my-plugin
cd my-plugin

lore-plugin test
lore-plugin build
```

Internally, build with standard Go:

```sh
GOOS=wasip1 GOARCH=wasm ...
```

The SDK should hide:

```text
WASM memory handling
ABI details
host-call transport
serialization
```

Plugin code should look conceptually like:

```go
plugin.RegisterMacro(...)
plugin.Pages().Search(...)
plugin.Settings().Get(...)
```

Standard Go should be the default compiler.

TinyGo may be supported later as an optional smaller-output compiler.

The generated package should be:

```text
dist/my-plugin.loreplugin
```

Suggested commit:

```text
feat: add plugin development sdk
```

---

# Phase 10 — Optional community registry

Only after runtime plugins are stable.

Possible future workflow:

```text
Administration
→ Plugins
→ Browse plugins
```

or:

```sh
lore plugin install github
```

A registry should contain metadata and downloadable signed/checksummed `.loreplugin` packages.

Do not build this during the core refactor.

---

# Security requirements throughout all phases

Treat community plugins as untrusted.

Do not allow:

```text
arbitrary native code inside Lore
direct database access
unrestricted filesystem access
unrestricted environment access
subprocess execution
unrestricted outbound network access
sanitizer bypass
direct access to Lore internal packages
```

A broken plugin must not crash Lore.

Design the runtime so resource limits can be added if they are not implemented immediately.

The sanitizer remains a trusted Lore-core boundary.

---

# Cross-cutting requirements

Rendering behavior must remain consistent across:

```text
normal page rendering
editor preview
export/PDF rendering
static-site generation
```

These paths should consume the same plugin/render registry.

Keep package boundaries explicit.

Prefer small interfaces.

Avoid switches on known plugin IDs where a registry/module mechanism is appropriate.

Do not build parallel implementations for bundled and installed plugins.

Keep every phase independently buildable and testable.

---

# Recommended Codex execution plan

Do not ask Codex to execute all phases in one run.

Use approximately:

```text
Run 1 → Phase 1
Run 2 → Phase 2
Run 3 → Phase 3
Run 4 → Phase 4
Run 5 → Phase 5
Run 6 → Phase 6
Run 7 → Phase 7
Run 8+ → remaining migrations/tooling
```

Phase 1 should establish the architecture.

Phase 2 should prove that the architecture survives a WASM boundary.

Phase 3 should prove that useful features can work without Lore-internal access.

Tables in Phase 6 should be treated as the final architecture stress test.

The key final success criterion is:

> A bundled Lore plugin and a user-installed copy of the same `.loreplugin` package are loaded through the same manager, WASM runtime, Plugin API, registry, permissions, storage, rendering pipeline, and asset mechanism, while the user-installed version can be installed, disabled, upgraded, and removed without rebuilding or restarting Lore.

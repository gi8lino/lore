# Lore Go Plugin SDK

`pluginsdk` is the public Go guest API. It has no dependencies on Lore internal
packages. All first-party guests use it, and manually installed packages use the
same manager, WASI runtime, permissions and sanitizer.

Build the CLI from a Lore checkout:

```sh
go build -o /tmp/lore-plugin ./cmd/lore-plugin
/tmp/lore-plugin init my-plugin
cd my-plugin
/tmp/lore-plugin test
/tmp/lore-plugin build
```

The result is `dist/my-plugin.loreplugin`. Upload it in Administration → Plugins.
For a published SDK version, install `github.com/gi8lino/lore/cmd/lore-plugin` at
that version and use `lore-plugin init my-plugin`. The CLI pins its own module
version in the generated project. `--sdk-version` selects another published
version; a checkout-built CLI uses its source checkout when available. Use `--sdk-path` to explicitly select a checkout (including when building
the CLI with trimpath). Development checkout selection writes a local Go module replacement; remove it and pin the released version before sharing.

## Ordinary Go contributions

Register contributions in `init`, not `main`: Lore initializes a WASI reactor
without running the program entry point. No ABI file or generated shim is needed.

```go
package main

import (
    "strings"
    plugin "github.com/gi8lino/lore/pluginsdk"
)

func main() {}

func init() {
    plugin.RegisterMacro("hello", func(source string) (struct{}, bool) {
        return struct{}{}, strings.TrimSpace(source) == "{{hello}}"
    }, func(_ struct{}) (plugin.Result, error) {
        return plugin.Text("<p>Hello!</p>"), nil
    })
}
```

Declare a `macro` module with ID and name `hello` in `plugin.yaml`.
`RegisterMacro` handles parsing stages and argument serialization. Lore protects
fenced code before dispatching macros. Macro results are HTML text parts.
`RegisterModule` supports renderer extensions and code highlighters using typed
`Request`/`Result` values. Register each executable manifest module explicitly.
Declarative syntax, style, settings, browser and resource modules are declared in
the manifest; they need no extra guest handlers. Executable modules may also
declare optional `usage` selectors (`contains`, `macro`, `substitution`, or
`fence`) so Lore can skip the guest for pages that cannot use the module. Treat
these as conservative performance hints: a false positive only does extra work,
but a false negative would suppress required rendering. Modules without usage
selectors remain always active. Macro registrations are automatically source-aware
from the manifest macro name.

`Text` returns intermediate output, `Markdown` asks Lore to render a nested
fragment (preprocessors only), and `Failure` returns an error. Every HTML result
still crosses the central sanitizer. Registration happens at initialization;
handlers should not mutate registrations while rendering. Panics become a
bounded generic error; traps and deadlines remain isolated by the host runtime.

## Typed capabilities

```go
page, err := plugin.Pages().Get("handbook")
pages, err := plugin.Pages().Search(plugin.PageQuery{Query: "tag:guide", Limit: 20})
content, err := plugin.Pages().Content("handbook")
value, err := plugin.Settings().Get("theme")
err = plugin.Settings().Set("theme", []byte("dark"))
value, err = plugin.Storage().Get("cache")
err = plugin.Storage().Set("cache", []byte("data"))
err = plugin.Log("render completed")
```

`StoredValue.Found` distinguishes a missing key from an empty value. Navigation,
attachment ranges and icon rendering also have typed helpers. Declare required
permissions in the manifest; the host checks both declarations and grants on
every call. Page access is viewer-scoped, storage is plugin-namespaced, and SDK
clients cannot choose a different plugin identity. `NewClient(transport)` accepts
a fake transport for ordinary native unit tests; the default transport calls Lore.

The guest has no ambient filesystem, environment, database, subprocess or network
capabilities. Importing a host development package does not create a WASI grant.
The SDK exposes no sanitizer policy or trusted-HTML bypass.

## Build contract

`test` validates the runtime manifest schema, project files and assets, then runs
`go test ./...` when the project has a Go module. `build` performs the same
validation and writes a bounded ZIP.
When the manifest contains executable renderer extensions, macros, or code
highlighters it compiles standard Go (`GOOS=wasip1 GOARCH=wasm`,
`-buildmode=c-shared`) and includes `plugin.wasm`. Declarative-only packages skip
the Go compiler and do not require a `go.mod`. Executable guests use the exact Go
version from `go.mod`; trimpath, stripped build IDs, sorted entries and fixed ZIP
metadata make repeated builds deterministic. Pin module versions and precompile
browser sources for reproducible inputs.

`pluginpackage` is the shared public manifest/package validator used by both Lore
and the CLI. `pluginsdk/build` is host-side developer tooling; it is not imported
by guests. All `assets/` files are packaged, including JS/CSS dependencies;
manifest-referenced files must exist. Symlinks, traversal, special files,
unsupported manifests and oversized packages are rejected. Browser TypeScript
compilation is project-owned; the CLI does not install a frontend toolchain.

The low-level [wire contract](../pluginapi/README.md) remains available for other
languages and advanced integrations. Go authors need not implement its exports,
allocation, host-call envelopes or macro serialization.

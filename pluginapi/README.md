# Lore render ABI v1

This is the small wire contract for Phase 2, not the full developer SDK. The `pluginapi` package has no dependency on Lore internals. Plugins may be written in any language that can produce a WASI Preview 1 reactor with these exports:

| Export             | Parameters                      | Result                         |
| ------------------ | ------------------------------- | ------------------------------ |
| `_initialize`      | none                            | none                           |
| `lore_api_version` | none                            | `i32`, currently `1`           |
| `lore_alloc`       | `i32` byte length               | `i32` offset into guest memory |
| `lore_transform`   | `i32` offset, `i32` byte length | `i64` packed response          |
| `memory`           | —                               | linear memory                  |

Lore calls `_initialize` before checking the ABI version. For each request it asks the guest to allocate a buffer, writes a JSON `RenderRequest`, and invokes `lore_transform`. The returned `i64` packs the response byte length into its high 32 bits and its guest-memory offset into the low 32 bits. Lore validates both before decoding JSON. The response buffer must remain valid until the next invocation. The host serializes calls to each reactor.

A renderer module receives its manifest module ID, stage (`preprocess` or `postprocess`), source, and presentation feature flags. A `RenderResult` contains an error or ordered fragments. A fragment is literal intermediate text, or Markdown for the host to render recursively. Only preprocessors may return Markdown fragments. The WASM call completes before the host renders those fragments; nested rendering therefore does not re-enter an executing Go guest.

All resulting HTML goes through Lore's central sanitizer. There is no trusted HTML result type. The guest cannot change sanitizer policy or receive request macro callbacks, filesystem handles, database clients, or Lore Go pointers. Unknown JSON fields, trailing JSON, invalid memory ranges, and excessive output are errors.

The bundled example in `plugins/callouts` is an independent Go module. Its `main.go` owns callout parsing; `abi_wasm.go` contains the small amount of memory transport code that a later SDK will hide. Standard Go builds it with:

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
```

Go's reactor and export behavior is documented in [Extensible Wasm Applications with Go](https://go.dev/blog/wasmexport). No TinyGo, C compiler, external WASM process, or native plugin loading is required.

Page/search capabilities, permission-checked host calls, and namespaced storage are reserved for Phase 3. Public installation and persistent enable/disable state are reserved for Phase 4. Browser asset serving remains Phase 5.

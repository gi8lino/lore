# Tables

This package owns `{table ...}` preprocessing, HTML presentation, browser
sorting/filtering, CSS, and declarative settings dependencies. The WASM guest has
no internal Lore imports or ambient host capabilities. `browser:render` is the
only requested permission.

`markdown-syntax` selects a standard core grammar through the public manifest
API. This keeps inline links, variables, references, and other registered syntax
in the same Goldmark parse. Core provides these grammars equally to community
packages; the table package owns activation and the central renderer has no
table-specific branch. Custom preprocessors/postprocessors execute in WASM.

Plain tables remain semantic HTML in Lore's document. Opted-in sorting/filtering
runs on a sanitized copy in an opaque frame. The original remains the fallback
for scripts being unavailable, errors, printing, and plugin disable. Core
filters the package stylesheet to scoped color declarations for that fallback;
all other CSS stays inside the isolated frame.

`browser.ts` is compiled separately into `assets/plugin.js` by
`make plugin-packages`; it never enters Lore's main frontend bundle. The pinned
TypeScript toolchain is installed with `npm ci`. The guest builds with standard
Go WASI. Runtime-installed copies use these exact generated package bytes.

Existing rendering preferences remain compatibility inputs until plugin
administration is migrated. Their dependency rules come from this manifest.

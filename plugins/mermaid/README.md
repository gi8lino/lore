# Mermaid plugin

A standard-Go WASI postprocessor marks Mermaid fences for the public sandboxed
browser-module API. Assets ship inside the same package for bundled and manual
installation. Core has no Mermaid browser loader.

`assets/mermaid.min.js` is the pinned Mermaid 12.0.0 distribution previously
vendored by Lore; its MIT license is in `assets/LICENSE`. Update it explicitly
with `scripts/web/vendor-mermaid.sh`, then run `make plugin-packages` to regenerate
the embedded archive. Ordinary frontend builds do not download or bundle it.

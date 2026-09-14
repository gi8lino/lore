# Bundled Chroma lexers

These lexer definitions are selected from Chroma v2.27.0 and redistributed under the MIT terms in `LICENSE`.

Lore intentionally vendors only the definitions used by its bundled syntax-highlighting plugin so the WASM guest does not link Chroma's complete global lexer catalogue. The local Go lexer in `../go_lexer.go` is adapted from Chroma's Go lexer for the same reason.

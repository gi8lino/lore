# Syntax Highlighting

Syntax Highlighting highlights fenced code blocks using Chroma and the explicit Markdown fence language supplied by Lore.

## Usage

````markdown
```go
func main() {
    println("Lore")
}
```
````

The plugin uses Chroma's complete maintained lexer registry rather than a curated subset, so every language and alias shipped by the pinned Chroma version is available.

Lore already knows the language from the fenced code block and passes it directly to the plugin; the plugin accepts Chroma language names and aliases and does not auto-detect from source text or treat the fence as a filename. Pages without fenced code do not select this module during rendering. Unsupported fence languages return unmatched and fall back to an ordinary fenced code block.

This bundled plugin is Lore's default provider for the exclusive `code-highlighter` contribution. Disable it before enabling another plugin that provides its own highlighter. Lore itself does not depend on Chroma once this plugin is disabled.

The plugin owns its token markup and presentation stylesheet. Lore sanitizes the returned HTML and filters/scopes the stylesheet before either reaches rendered page content.

## Permissions

This plugin requests no Lore capabilities.

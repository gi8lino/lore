# Syntax Highlighting

Syntax Highlighting highlights fenced code blocks that name a language supported by Chroma.

## Usage

````markdown
```go
func main() {
    println("Lore")
}
```
````

This bundled plugin is Lore's default provider for the exclusive `code-highlighter` contribution. Disable it before enabling another plugin that provides its own highlighter, such as a Prism-, Shiki-, or tree-sitter-based implementation. Lore itself does not depend on Chroma once this plugin is disabled.

The plugin owns its token markup and presentation stylesheet. Lore sanitizes the returned HTML and filters/scopes the stylesheet before either reaches rendered page content. Unsupported languages fall back to an ordinary fenced code block.

## Permissions

This plugin requests no Lore capabilities.

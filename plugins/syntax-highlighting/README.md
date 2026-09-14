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

The plugin exposes a curated runtime registry of common languages while reusing Chroma's maintained lexer definitions and aliases. The current set covers Bash/shell, C/C++/C#, CSS, Dockerfile, Go, HCL, HTML, Java, JavaScript, JSON, Kotlin, Lua, Makefile, Markdown, Nginx, PHP, PowerShell, Python, Ruby, Rust, SQL, Terraform, TOML, TypeScript, XML, YAML, and Zig.

Lore already knows the language from the fenced code block and passes it directly to the plugin; the plugin does not auto-detect the language from the source. Pages without fenced code do not select this WASM module at all. Unsupported fence languages return unmatched and fall back to an ordinary fenced code block.

The selected Chroma lexer objects are re-registered into a small registry. This keeps alias and nested sublexer lookup bounded to the curated set while avoiding copied lexer definitions. Chroma still compiles regex rules lazily on first use and retains them on the lexer instances for subsequent blocks.

This bundled plugin is Lore's default provider for the exclusive `code-highlighter` contribution. Disable it before enabling another plugin that provides its own highlighter, such as a Prism-, Shiki-, or tree-sitter-based implementation. Lore itself does not depend on Chroma once this plugin is disabled.

The plugin owns its token markup and presentation stylesheet. Lore sanitizes the returned HTML and filters/scopes the stylesheet before either reaches rendered page content.

## Permissions

This plugin requests no Lore capabilities.

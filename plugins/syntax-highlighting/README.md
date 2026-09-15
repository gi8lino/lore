# Syntax Highlighting

Syntax Highlighting highlights fenced code blocks using a small deterministic scanner and the explicit Markdown fence language supplied by Lore.

## Usage

````markdown
```go
func main() {
    println("Lore")
}
```
````

The plugin supports the same common wiki, application, infrastructure, and configuration fence names as the previous bundled highlighter: Bash/shell, C/C++/C#, CSS, Dockerfile, Go, HCL/Terraform, HTML/XML, Java, JavaScript/TypeScript, JSON, Kotlin, Lua, Makefile, Markdown, Nginx, PHP, PowerShell, Python, Ruby, Rust, SQL, TOML, YAML, and Zig.

Lore already knows the language from the fenced code block and passes it directly to the plugin. The plugin never guesses a language from source text. Unsupported fence languages return unmatched and fall back to an ordinary fenced code block.

Tokenization uses direct byte/rune scanning, fixed keyword tables, and small language profiles. It does not use regular expressions and has no syntax-highlighting dependency. Related languages share scanners where their lexical rules are compatible.

The `chroma` module ID and `.chroma` wrapper class are retained as stable output identifiers so the plugin remains a drop-in replacement for the existing Lore renderer and stylesheet contract. They do not imply a Chroma runtime dependency.

The plugin owns its token markup and presentation stylesheet. Lore sanitizes the returned HTML and filters/scopes the stylesheet before either reaches rendered page content.

## Permissions

This plugin requests no Lore capabilities.

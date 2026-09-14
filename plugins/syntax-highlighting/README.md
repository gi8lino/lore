# Syntax Highlighting

Syntax Highlighting applies server-side highlighting to fenced code blocks that name a supported language.

## Usage

````markdown
```go
func main() {
    println("Lore")
}
```
````

Lore's bundled implementation uses Chroma and emits stable token classes so themes can style highlighted code. Disable this plugin to render ordinary fenced code instead, or install a different rendering plugin that provides another highlighting strategy.

## Permissions

This plugin requests no Lore capabilities. It activates Lore's public `syntax-highlighting` render policy.

# Typographer

Typographer converts common ASCII punctuation sequences into typographic characters while Lore renders Markdown.

## Example

```markdown
"Lore" -- documentation...
```

Depending on the text, Typographer can produce curly quotes, typographic dashes, and ellipses. The plugin is disabled by default because these substitutions intentionally change rendered punctuation.

When **Coding Ligatures** is enabled as well, Lore preserves programming-oriented operator sequences such as `-->`, `<<`, and `>>` so the ligature font can render them instead of Typographer consuming them first.

## Permissions

This plugin requests no Lore capabilities. It activates Lore's public `typographer` render policy.

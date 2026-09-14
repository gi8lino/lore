# Markdown helpers

`plugins/markdown` contains small Markdown parsing helpers shared by Lore core and plugin implementations.

This directory is not a Lore plugin. It has no `plugin.yaml`, is not packaged as a `.loreplugin`, and does not appear in the plugin administration UI.

## Fenced code blocks

`fences.go` handles Markdown fenced code-block boundaries. Lore uses these helpers when a feature needs to recognize code blocks without interpreting their contents.

For example, content inside a fenced block must remain literal instead of being processed as wiki links, macros, directives, or other Markdown extensions:

````markdown
```text
{{subpages}}
[[Some Page]]
```
````

The package exposes:

- `Fence` to recognize an opening backtick or tilde fence and return its complete marker;
- `Closes` to determine whether a line closes a previously opened fence;
- `AppendFence` to copy a complete fenced block without interpreting its body.

Keep this package focused on reusable Markdown helpers needed by both Lore and standalone plugin modules. Plugin-specific parsing belongs in the plugin that owns the syntax.

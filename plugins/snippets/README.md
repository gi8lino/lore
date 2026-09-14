# Snippets

Snippets provides reusable Markdown blocks through `{{snippet:name}}` macros.

## Usage

Create snippets from **Administration → Plugins → Snippets**. Each snippet has a stable name, optional description, and Markdown content.

```markdown
{{snippet:deployment-warning}}
```

The editor includes stored snippets in `{{` completion and slash-command results.

Snippet content is inserted once and is not recursively evaluated as more plugin macros. This keeps reusable content predictable and prevents a stored snippet from unexpectedly invoking Variables, Includes, or another content plugin.

Fenced code blocks keep snippet syntax literal.

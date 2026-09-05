# Markdown

Lore uses Goldmark with optional extensions controlled by administrators. The default renderer enables wiki links, callouts, tabs, details, tables, table styling/sorting/filtering, strikethrough, task lists, autolinks, syntax highlighting, footnotes, definition lists, and typographic substitutions.

## Wiki links

```markdown
[[Postgres Restore]]
[[Postgres Restore|the runbook]]
[[operations/postgres/restore]]
```

Wiki links are ignored inside fenced code blocks.

## Callouts

```markdown
!!! warning
`$(VAR_NAME)` does not work with **envFrom**!
```

Supported presentation kinds include `info`, `success`, `warning`, and `danger`.

## Tabs

```markdown
=== "Linux"

    ```bash
    apt install postgresql
    ```

=== "macOS"

    ```bash
    brew install postgresql
    ```
```

## Collapsible details

```markdown
??? "Closed by default"

    Markdown content.
```

Use `???+` to render the details block initially open.

## Page functions

A standalone function can insert the current page's child navigation:

```markdown
{{subpages}}
```

The static filesystem builder supports `{{subpages}}` as well.

## Server-only knowledge macros

The PostgreSQL-backed wiki can expand reusable knowledge content before Markdown rendering:

```text
{{var:name}}
{{snippet:name}}
{{include:path/to/page}}
```

Variables and snippets come from the knowledge-snippet store; includes insert another page's Markdown. Expansion is skipped inside fenced code and recursion is bounded. Filesystem static mode does not depend on the database, so these database-backed macros are not expanded there.

See [Tables](tables.md) for Lore's table directive syntax.

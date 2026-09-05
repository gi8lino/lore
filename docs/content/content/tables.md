# Tables

Lore renders GitHub-flavored Markdown tables. An optional directive immediately after a table can add theme-aware colors, sorting, and filtering.

```markdown
| Service | Status | Owner |
| --- | --- | --- |
| API | Healthy | Platform |
| DB | Warning | Data |

{table header=accent col:2=info row:2=warning cell:2,2=danger sortable filterable}
```

Rows and columns in directives are one-based. `row:N` addresses a body row, `col:N` or `column:N` addresses a column, and `cell:R,C` addresses one body cell.

Supported tones are:

```text
accent accent-soft info success warning danger neutral
gray blue purple green yellow orange red
```

`sortable` enables client-side column sorting. `filterable` adds per-column filtering controls. Administrators can disable table rendering, styling, sorting, or filtering independently.

The editor's table tools can apply the same directive-based formatting without requiring authors to memorize the syntax.

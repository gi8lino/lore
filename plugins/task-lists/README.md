# Task Lists

Task Lists renders GitHub-style Markdown task items with accessible visible checkbox state.

## Usage

```markdown
- [x] Create backup
- [ ] Run upgrade
```

The rendered checkboxes are presentation only; editing the Markdown remains the way to change their state.

## Permissions

This plugin requests no Lore capabilities. It selects Lore's public `task-list` Markdown grammar. The host implementation of that grammar includes the inert accessible checkbox renderer, so disabling the plugin removes both task-list parsing and task-list presentation from new renders.



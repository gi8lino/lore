# Templates and snippets

Lore has two reusable-content mechanisms in the server application.

## Page templates

Administrators create named page templates with a description and Markdown body. Authors can choose a template when creating a new page to prefill editor content.

## Knowledge snippets

Administrators can store reusable knowledge items. Variables and Markdown snippets are referenced by name during page rendering:

```text
{{var:name}}
{{snippet:name}}
```

A page can also include another page's Markdown:

```text
{{include:platform/shared-warning}}
```

Includes and snippets can nest, with recursion protection and a maximum expansion depth. These macros are database-backed and therefore belong to normal server mode rather than filesystem static mode.

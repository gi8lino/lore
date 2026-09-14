# Variables

Variables provides reusable text values for Lore Markdown.

## Usage

Create values in **Administration → Plugins → Variables** and reference them with:

```text
{{var:environment}}
```

Variable names are case-insensitive when referenced. The stored name remains the canonical name shown in the editor, page inspector, and export controls.

## Editor integration

Type `{{` to search variables, use **Insert → Variable**, or select a variable from slash commands. The editor inserts the canonical `{{var:name}}` macro.

Macros inside fenced code blocks remain literal.

## Page inspection

Pages that use variables expose a Variables inspector. It lists each distinct variable, its saved value and description, and the number of occurrences. Highlighting marks only text originating from a variable; ordinary text with the same value is not marked.

## Print and PDF overrides

The Share and export dialog lets readers temporarily override variables used by the page. Overrides apply only to that export request, may intentionally be empty, and never update the saved variable or page source.

Inserted values are not recursively evaluated as new Lore macros.

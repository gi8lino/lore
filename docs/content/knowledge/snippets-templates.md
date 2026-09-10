# Templates and snippets

Lore has two reusable-content mechanisms in the server application.

## Page templates

Administrators create named page blueprints. A blueprint can prefill the page path prefix, icon, tags, lifecycle status, owner group, review interval, structured properties, and Markdown body. Authors can choose a blueprint when creating a page.

Blueprints can also declare prompted fields. Each field has a stable name, label, optional default, and optional required flag. Use `{{field:name}}` in the blueprint Markdown; the new-page editor asks for the value, renders it in live preview, and replaces the placeholder when the page is saved. Blueprint fields are creation-time inputs rather than persistent macros.

For example, a service blueprint can provide fields such as `service`, `repository`, and `tier`, pre-assign an owner and `service` tag, and seed properties used by dynamic page reports.

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

Includes can nest, with recursion protection and a maximum expansion depth. Stored variable and snippet values are inserted without recursively evaluating new knowledge macros. These macros are database-backed and therefore belong to normal server mode rather than filesystem static mode.

## Inspect variables on a page

Resolved values look like ordinary text while reading. A **Variables (N)** button
appears in the page actions only when the page expands variables. The count is
for distinct stored variables, not occurrences; differently cased references to
the same variable share one entry. Variables in included pages are counted too.

Open the panel to see each variable's name, saved value, description, and number
of occurrences. **Highlight in page** adds a subtle theme-aware background and a
dotted underline to the text originating from a variable. Hover, focus, or tap a
highlighted value to inspect its name and saved value. Ordinary text that happens
to match the saved value is not highlighted. Turn highlighting off to follow a
link whose label contains a highlighted variable.

Closing the panel leaves highlighting active until it is switched off or the
page is reloaded. Highlighting never appears in print or PDF output. The panel
is read-only: changing shared values remains an administrator action.

Variables used only in link destinations, image attributes, or complex Markdown
are still listed, even when there is no text that can be highlighted safely.
Lore keeps the normal rendered document whenever annotations would change its
content or structure.

## Temporary values for print and PDF

Open **Share and export**, then expand **Customize variables**. Each field starts
with its saved value. Change only the fields needed for this output, then use
**Preview print / PDF**, **Print**, or **Export PDF**. An intentionally empty field
replaces that variable with an empty string; surrounding whitespace is retained.
Every occurrence of the same variable receives the same temporary value.

The preview and browser print document use the PDF export's HTML and stylesheet.
The preview shows the content, not a pixel-identical simulation of pagination;
the browser and PDF service finalize page breaks separately. Like PDF export,
the preview uses server-rendered content rather than the reading page's live
filters or other browser-only enhancements. Browser print does
not require a configured PDF service.

**Reset to saved values** restores the fields and clears the preview. Closing and
reopening the dialog starts a fresh session. Overrides are held only for the
current dialog/request: they do not update shared variables, page Markdown,
revisions, local storage, or another reader's page. A normal browser print action
outside this dialog prints the reading page with its saved values. Use the
**Print** action in the dialog to print customized values.

Markdown exports remain original source, including macro syntax. This feature
does not change macro evaluation: fenced code stays literal, inline code can
expand variables, and includes retain their existing recursion limits. Stored
snippet contents and temporary values are not recursively evaluated as new
knowledge macros.

Temporary values follow the existing Markdown rendering and sanitization rules.
Export requests accept at most 128 changed variables, 8 KiB per value, and 64 KiB
in total for names and values. Only variables actually used by the page may be
overridden. None of these operations writes to the database.



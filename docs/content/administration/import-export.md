# Import and export

## Imports

The administrator import workspace accepts an explicitly selected source format. The import request limit is 100 MiB. Supported file contents also share a 100 MiB uncompressed budget across all files and ZIP archives in the request.

**Markdown** accepts `.md`, `.markdown`, or ZIP archives containing those files. Each document requires a level-one heading (`# Title`), which becomes the page title; its archive/file path becomes the page path.

**Wiki.js** accepts JSON or ZIP. JSON pages provide `path`, `title`, and `content` fields.

**Confluence** accepts HTML/HTM or ZIP and converts a supported HTML subset to Markdown before import.

Imported pages are created as `verified`. When an import replaces an existing path, Lore retains the existing page's metadata such as icon, language, tags, lifecycle state, owner/review settings, groups, deprecated target, and properties.

## Exports

A single page can be exported as Markdown. If it references stored images, Lore creates a ZIP containing the Markdown plus each referenced image and rewrites image paths to archive-relative locations.

Administrators can export selected pages or all pages as an archive. PDF export uses the normal Markdown renderer, inlines stored images, and sends the self-contained HTML to the configured HTML-to-PDF service.

Filesystem static site generation is a separate publishing path described in [Static sites](../static-sites.md).

## Customized print and PDF output

Readers can expand **Customize variables** in a page's **Share and export** dialog
to supply temporary values for that output. **Preview print / PDF** and **Print**
use the same resolved, image-embedded HTML and stylesheet as PDF export. PDF
conversion still happens in the configured HTML-to-PDF service, which receives
ordinary HTML and needs no variable-specific changes.

Overrides are request-local and discarded when the dialog closes. They never
change stored values, pages, revisions, or Markdown exports. See
[Templates and snippets](../knowledge/snippets-templates.md#temporary-values-for-print-and-pdf)
for the reading panel, reset behavior, and limits.

The browser-authenticated export endpoints accept JSON with the exact stored
variable names. Only changed values need to be supplied:

```json
{ "variables": { "environment": "staging" } }
```

`POST /export/preview/{slug...}` returns `{"document":"..."}` containing the clean
print document. `POST /export/pdf/{slug...}` returns the generated PDF. Both use
private, non-cacheable responses. Malformed requests return 400; invalid or unused
variable overrides return 422 through Lore's field-problem format. Existing
`GET /export/pdf/{slug...}` links continue to use saved values; URL parameters do
not set overrides. These are browser export routes, not bearer-token API routes.

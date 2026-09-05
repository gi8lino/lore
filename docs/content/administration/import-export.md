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

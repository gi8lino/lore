# Media and attachments

Lore stores uploaded media in PostgreSQL and exposes stable authenticated URLs.

## Images

Image uploads support JPEG, PNG, GIF, and WebP. Lore validates size/type before storage and tracks how many page references point to each image. Images can be deleted when policy allows it; in-use media is protected by reference checks.

## Attachments

Attachments are non-image documentation files. Supported types include PDF, plain text, Markdown, JSON, YAML, CSV, log files, TOML, and ZIP. Attachment metadata includes filename, MIME type, size, uploader, creation time, and usage count.

The configured limits are 10 MiB for images and 25 MiB for attachments.

## Export behavior

Markdown export includes referenced images. A page with no referenced images can be returned as a plain `.md` file; pages requiring media are packaged into a ZIP with rewritten relative media paths.

PDF export renders page HTML, inlines stored images as data URLs, and sends the self-contained document to the configured HTML-to-PDF service.

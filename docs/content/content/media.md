# Media and attachments

Lore stores uploaded media in PostgreSQL and exposes stable authenticated URLs.

## Images

Image uploads support JPEG, PNG, GIF, and WebP. Lore validates size/type before storage and tracks how many page references point to each image. Images can be deleted when policy allows it; in-use media is protected by reference checks.

### Image sizing

Append a width directive directly to a Markdown image, without a space:

```markdown
![Architecture](/media/42/architecture.png){width=640}
![Architecture](/media/42/architecture.png){width=640px}
![Architecture](/media/42/architecture.png){width=50%}
```

A bare number means pixels. Pixel widths accept whole numbers from 1 to 10000;
percentages accept whole numbers from 1 to 100. A percentage is relative to the
containing content area, not the original image dimensions.

Only the displayed width changes. The height follows the original aspect ratio,
and images remain constrained to the available content width on smaller screens.
The original upload and download size are unchanged. Remove the directive to
restore the normal image size.

Titles, links around images, and reference-style images also work:

```markdown
![Architecture](/media/42/architecture.png "System overview"){width=640}
[![Architecture](/media/42/architecture.png){width=50%}](/media/42/architecture.png)
![Architecture][diagram]{width=50%}

[diagram]: /media/42/architecture.png "System overview"
```

Sizing uses the shared renderer for editor previews, saved pages, and static
sites. PDF export retains the width, and Markdown exports keep the directive.
The editor's **Format** action preserves it as well.

This is a Lore Markdown extension, not standard Markdown image syntax. Other
Markdown renderers may show the directive as text. Width is the only supported
attribute: no height, arbitrary style, or extra attributes are accepted. Invalid
or unsupported directives remain visible text. Sizing examples inside code
spans or code blocks are not interpreted.

## Attachments

Attachments are non-image documentation files. Supported types include PDF, plain text, Markdown, JSON, YAML, CSV, log files, TOML, and ZIP. Attachment metadata includes filename, MIME type, size, uploader, creation time, and usage count.

The configured limits are 10 MiB for images and 25 MiB for attachments.

## Export behavior

Markdown export includes referenced images. A page with no referenced images can be returned as a plain `.md` file; pages requiring media are packaged into a ZIP with rewritten relative media paths.

PDF export renders page HTML, inlines stored images as data URLs, and sends the self-contained document to the configured HTML-to-PDF service.

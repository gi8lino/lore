# Static sites

Lore can work like a small MkDocs-style generator: write ordinary Markdown files in a directory and build a complete read-only site without PostgreSQL or a running Lore instance.

This repository's own documentation is configured by `lore-site.toml` and lives under `docs/`.

## Build

A built Lore binary already contains the read-only browser assets needed by the generator, so it can build a site directly:

```sh
lore build
```

From the source repository the convenience target builds the frontend first and then runs the generator:

```sh
make site
```

The default configuration file is optional. Without it, Lore uses `docs` as the source directory and `site` as the output directory. Command-line flags can override the configuration.

## Configuration

```toml
site_name = "Lore"
site_url = "https://gi8lino.github.io/lore/"
source_dir = "docs"
output_dir = "site"
theme = "Light"
language = "en"
mermaid = true
```

`site_url` determines the URL prefix used by generated links. This matters for project sites such as GitHub Pages, where Lore may be hosted below `/lore/` rather than at the domain root.

## Filesystem routes

Markdown paths map directly to clean static URLs. The source directory must contain a root `index.md`, which becomes the site home page:

```text
docs/index.md                   -> /
docs/getting-started.md         -> /getting-started/
docs/installation/index.md      -> /installation/
docs/installation/docker.md     -> /installation/docker/
```

When `site_url` contains a path prefix, that prefix is prepended to every generated URL.

## Links and assets

Ordinary relative Markdown links are supported:

```markdown
[Docker](installation/docker.md)
```

Lore resolves the source file at build time and rewrites the link to the generated HTML route. A `.md` link to a missing source file fails the build.

Non-Markdown files under the source directory are copied into the output tree. Relative image and asset URLs are rewritten so they continue to work after page routes become directory-style URLs.

Lore wiki links use the same Lore renderer and are rewritten to static routes. Unresolved or ambiguous wiki-link targets fail the build, so a published static site does not silently ship broken Lore links. `{{subpages}}` is generated from the filesystem page hierarchy.

## What the build contains

A static build includes:

- generated HTML pages and a `404.html` page;
- Lore's CSS and the selected theme data;
- only the read-only TypeScript modules needed for navigation, page contents, Markdown enhancements, and static search;
- `search-index.json` for browser-side search;
- source assets such as images;
- `.nojekyll` for GitHub Pages;
- `sitemap.xml` when `site_url` is an absolute HTTP(S) URL.

It intentionally does **not** ship the Lore editor, authentication, account menus, admin UI, drafts, notifications, API tokens, or write APIs. The output is ordinary static files and can be hosted by GitHub Pages, Cloudflare Pages, S3-compatible storage, or any web server.

## GitHub Pages

A typical CI job builds the frontend and Lore binary, runs `lore build`, and publishes the generated `site/` directory as the Pages artifact. No PostgreSQL service is needed for that job.

For local preview, override the site URL to match your local server root if the checked-in configuration uses a GitHub Pages project prefix.

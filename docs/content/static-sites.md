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

Lore wiki links use the same Lore renderer and are rewritten to static routes. Unresolved or ambiguous wiki-link targets fail the build, so a published static site does not silently ship broken Lore links. `{{subpages}}` is generated from the filesystem page hierarchy and supports the same optional `title="..."` heading override as server-rendered pages.

## Logos, favicons, and extra assets

Optional settings select your site's branding and an extra asset directory:

```toml
# Paths are relative to this lore-site.toml file.
logo = "assets/logo.svg"
favicon = "assets/favicon.svg"
favicon_ico = "assets/favicon.ico"
assets_dir = "assets"
```

Absolute paths are also accepted. These four paths resolve relative to the configuration file; the existing `source_dir` and `output_dir` settings remain relative to the working directory.

| Setting       | Generated result                                                                                                  |
| ------------- | ----------------------------------------------------------------------------------------------------------------- |
| `logo`        | Header image at `assets/site-logo.<extension>`, with the site name as alternative text.                           |
| `favicon`     | Browser icon at `assets/site-favicon.<extension>`.                                                                |
| `favicon_ico` | ICO fallback at `favicon.ico` in the output root, also linked from each page.                                     |
| `assets_dir`  | Non-Markdown files copied recursively under `assets/`, preserving subdirectories and skipping hidden directories. |

Logo and favicon images support SVG, PNG, JPEG, WebP, GIF, and ICO. `favicon_ico` must point to an actual ICO image: Lore copies images without converting them. Missing configured paths, incorrect file/directory types, and paths overlapping the output directory fail validation before the output is cleared.

Omit `logo` or `favicon` to retain Lore's default branding. Omit `favicon_ico` to omit the fallback link, or `assets_dir` when all your assets already live under `source_dir`. Generated logo and favicon links include the `site_url` path prefix, including on search and error pages.

For example, with the configuration at `docs/lore-site.toml`, `logo = "assets/logo.svg"` reads `docs/assets/logo.svg`. A file `docs/assets/images/example.png` from that asset directory becomes `assets/images/example.png` in the generated site.

Extra assets are copied first, followed by Lore's bundled assets, files from `source_dir`, and explicitly configured branding images. Later copies take precedence. Avoid using Lore's `assets/js/` and `assets/css/` paths for your own files. The older `source_dir/assets/favicon.svg` override still works when `favicon` is omitted.

For images used inside Markdown, you can continue placing them under `source_dir` and linking with relative paths, such as `![Logo](images/logo.svg)` from the root `index.md`. Do not edit files directly in the output directory: each build deletes and recreates it.

Some browsers request `/favicon.ico` even when an icon is declared. The fallback file handles this when the site is served at the domain root. On a project site under `/never/`, an unsolicited request to `/favicon.ico` belongs to the host's root; the generated icon links correctly use `/never/`.

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

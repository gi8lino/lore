# Static sites

Lore can work like a small MkDocs-style generator: write ordinary Markdown files in a directory and build a complete read-only site without PostgreSQL or a running Lore instance.

This repository's own documentation is configured by `docs/site.toml`; the published Markdown lives under `docs/content/`.

## Build

A built Lore binary already contains the read-only browser assets needed by the generator, so it can build a site directly:

```sh
lore build
```

From the source repository the convenience target builds the frontend first and then runs the generator:

```sh
make site
```

The default `lore-site.toml` configuration file is optional. Without it, Lore uses `Documentation` as the site name, `docs` as the source directory, and `site` as the output directory. Command-line flags can override the configuration.

## Configuration

A complete `site.toml` can use all of the following settings:

```toml
site_name = "My Documentation"
site_url = "https://docs.example.com/"

source_dir = "content"
output_dir = "site"

theme = "Light"
language = "en"
mermaid = true

logo = "../branding/logo.svg"
favicon = "../branding/favicon.svg"
favicon_ico = "../branding/favicon.ico"
assets_dir = "../assets"

[[external_links]]
label = "Repository"
url = "https://github.com/example/project"
icon = "github-simple"
description = "v2.4.1"
```

`logo`, `favicon`, `favicon_ico`, and `assets_dir` are resolved relative to the configuration file. Normal relative paths, including `../`, are supported, so assets may live in a parent directory. Absolute paths are supported too. `source_dir` and `output_dir` are resolved relative to the process working directory.

`site_url` determines the URL prefix used by generated links. This matters for project sites such as GitHub Pages, where a site may be hosted below a repository path rather than at the domain root.

`external_links` adds optional links beside search in the generated header. Each entry requires `label` and an absolute HTTP(S) `url`; `icon` is an optional icon identifier and `description` is optional secondary text such as a version, environment, or provider name. Icon identifiers explicitly include their source: use `-lucide` for Lucide interface icons and `-simple` for Simple Icons brand logos, for example `book-open-lucide` or `github-simple`. Multiple entries are rendered in configuration order.

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

Branding is entirely opt-in. The builder does not copy Lore logos or favicons into a generated site.

```toml
# These paths are relative to the configuration file.
logo = "assets/logo.svg"
favicon = "content/assets/favicon.svg"
favicon_ico = "content/favicon.ico"
assets_dir = "assets"
```

These path-resolution rules are the same as in the complete configuration example above.

Configured branding files keep a natural public path instead of being renamed:

- a file below `source_dir` keeps its path relative to `source_dir`;
- a file below `assets_dir` is published below `assets/` with the same relative path;
- a branding file outside both trees is published below `assets/` using its own filename;
- `assets_dir` itself is copied recursively below `assets/`, preserving subdirectories and skipping hidden directories.

This repository opts into Lore branding explicitly: `docs/site.toml` points `logo` and `favicon` at `../web/src/lore.svg` and `../web/src/favicon.svg`. Because those files are outside both published source trees, they are copied to `assets/lore.svg` and `assets/favicon.svg`. A configured `content/favicon.ico` would instead resolve below `docs/content` and publish as `favicon.ico` at the site root.

Logo and favicon images support SVG, PNG, JPEG, WebP, GIF, and ICO. `favicon_ico` must point to an ICO file; Lore copies images without converting them. Missing paths, incorrect file/directory types, and paths overlapping `output_dir` fail validation before existing output is cleared.

When `logo` is omitted, the header displays `site_name` as text. When `favicon` or `favicon_ico` is omitted, the corresponding icon link is omitted. There is no implicit Lore branding fallback.

Build assets are copied in this order: configured `assets_dir`, the static browser runtime, non-Markdown files from `source_dir`, and explicit branding files. Later copies take precedence. Avoid placing user files at the runtime-owned `assets/js/` and `assets/css/` paths.

For images used inside Markdown, you can continue placing them under `source_dir` and linking with relative paths, such as `![Logo](images/logo.svg)` from the root `index.md`. Do not edit files directly in `output_dir`: each build deletes and recreates it.

## What the build contains

A static build includes:

- generated HTML pages and a `404.html` page;
- the static-site CSS runtime and selected theme data;
- only the read-only TypeScript modules needed for navigation, page contents, Markdown enhancements, and static search;
- no built-in logo, mark, or favicon files unless the user explicitly configures them;
- `search-index.json` for browser-side search;
- source assets such as images;
- `.nojekyll` for GitHub Pages;
- `sitemap.xml` when `site_url` is an absolute HTTP(S) URL.

It intentionally does **not** ship the Lore editor, authentication, account menus, admin UI, drafts, notifications, API tokens, or write APIs. The output is ordinary static files and can be hosted by GitHub Pages, Cloudflare Pages, S3-compatible storage, or any web server.

## GitHub Pages

A typical CI job builds the frontend and Lore binary, runs `lore build`, and publishes the generated `site/` directory as the Pages artifact. No PostgreSQL service is needed for that job.

For local preview, override the site URL to match your local server root if the checked-in configuration uses a GitHub Pages project prefix.

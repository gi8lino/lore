# Lore

Lore is a small, self-hosted, Markdown-first wiki. The normal server is a single Go binary backed by PostgreSQL; the same codebase can also build a read-only static documentation site from ordinary Markdown files.

## Choose a mode

**Lore server** is the full collaborative wiki. It provides authentication, roles, editing, private drafts, page history, groups, search, media, administration, API tokens, discussions, and PostgreSQL-backed knowledge features.

**Lore static site** reads Markdown from a directory and emits HTML, CSS, generated JavaScript, a navigation tree, page contents, and a browser-side search index. It does not use PostgreSQL and intentionally contains no authentication, editor, drafts, administration, or write APIs.

## Start here

- [Getting started](getting-started.md) walks through the fastest server setup.
- [Installation](installation/index.md) covers Docker and direct binary use.
- [Configuration](configuration/index.md) explains runtime and application settings.
- [Authentication](authentication/index.md) covers local, trusted-proxy, and OIDC modes.
- [Content](content/index.md) covers pages, Markdown, tables, media, lifecycle, and organization.
- [Knowledge tools](knowledge/index.md) covers search, links, graph, templates, and snippets.
- [Collaboration](collaboration/index.md) covers roles, groups, discussions, and notifications.
- [User experience](user/index.md) covers the dashboard, preferences, saved searches, and account tools.
- [Administration](administration/index.md) covers health, imports, exports, audit, and the recycle bin.
- [API](api/index.md) documents the authenticated JSON surface and personal access tokens.
- [Static sites](static-sites.md) documents the filesystem builder used for this documentation.
- [Development](development.md) and [architecture](architecture.md) describe the codebase.

{{subpages}}

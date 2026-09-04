# Lore

A small, self-hosted, Markdown-first wiki for documentation. The full server is a single Go binary backed by PostgreSQL, and the same binary can build read-only static sites directly from Markdown files without a database.

## Features

- Server-rendered, responsive interface with per-user file-based themes and optional page contents
- Live PostgreSQL full-text search with direct page navigation and `tag:`, `group:`, `title:`, `namespace:`, and `author:` filters
- Markdown, sortable/filterable GFM tables with theme-aware cell colors, syntax highlighting, callouts, tabs, collapsible details, Mermaid diagrams, images, and `[[Wiki Links]]`
- Automatic backlinks, persistent tag autocomplete, immutable revision history, favorites, recent views, and popular pages
- Private server-side editor drafts with browser fallback autosave, cross-device resume, and explicit discard controls
- JSON REST API and expiring personal access tokens with hashed-at-rest secrets
- Read-only public page permalinks with opaque hashed-at-rest bearer tokens
- OIDC (Authentik, Keycloak, Entra ID) with stable subject identities and optional mapped group sync, trusted-proxy headers, local accounts, or no-auth development mode
- Administrator console for users, groups, navigation icons, tags, tokens, exports, and image cleanup
- Embedded migrations and web assets; no Redis, Elasticsearch, or Node.js runtime. PDF export uses WeasyPrint on the server.

## Deployment

### Docker Compose

Use the [deploy/compose.yaml](deploy/compose.yaml) file to run Lore with Docker Compose.

Start Lore with:

```sh
docker compose -f deploy/compose.yaml up -d
```

Open `http://localhost:8080`.

On a fresh database, Lore redirects to `/setup` to create the first local administrator. New-user registration starts closed; configure the normal authentication mode afterwards in Administration. Set `LORE_LOCAL_LOGIN=true` to expose `/auth/local` alongside OIDC or trusted-proxy authentication as a break-glass login.

## Configuration

Run:

```sh
docker run --rm ghcr.io/gi8lino/lore:latest --help
```

for deployment-level configuration options. Non-secret browser authentication settings are managed in **Administration → Configuration** and take effect without restarting Lore.

OIDC identities are bound to the verified `(issuer, subject)` pair. Usernames, email addresses, and display names are profile attributes and never determine account ownership.

When automatic user creation is disabled, unknown verified OIDC identities are queued in **Administration → Users**. Administrators can create a new Lore user, link the new identity to an existing account after an identity-provider rebuild, or reject it.

OIDC keeps secrets outside PostgreSQL:

- `LORE_OIDC_CLIENT_SECRET` contains the provider client secret.
- `LORE_SESSION_SECRET` signs login sessions and must contain at least 32 characters.
- `LORE_AUTH_MODE` is an optional emergency override (`none`, `trusted-proxy`, or `oidc`) for recovering from an invalid database-managed auth configuration.

## Static documentation sites

Lore can also build a read-only documentation site directly from Markdown files without PostgreSQL, authentication, or a running Lore server. The repository's own documentation uses this mode:

```sh
make site
```

The default `lore-site.toml` reads Markdown from `docs/` and writes deployable HTML/CSS/JavaScript to `site/`. The generated site contains navigation, page contents, Lore Markdown rendering, and a client-side search index, but intentionally contains no editor, authentication, administration, drafts, API tokens, or other server-only features. It can be hosted on GitHub Pages or any static web host.

See [Static sites](docs/static-sites.md) for the filesystem layout, configuration, URL handling, and deployment model.

## Development

All authored frontend code is TypeScript. Browser modules and the service worker live in `web/src/ts`, frontend tests live in `test/ts`, and relative source imports use `.ts` extensions. `tsc` rewrites those imports and emits native browser JavaScript into `web/dist` during the build. The service worker and Node-based tests use dedicated TypeScript projects so their Web Worker and Node environments are typed independently. JavaScript exists only as generated browser output or external runtime assets; Node.js, npm, and TypeScript are build-time dependencies only, so the deployed application remains a single Go binary.

See [ARCHITECTURE.md](ARCHITECTURE.md) for package responsibilities and dependency boundaries.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.



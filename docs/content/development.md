# Development

Lore is a Go monolith with a server-rendered frontend enhanced by pure TypeScript and plain CSS.

## Common commands

```sh
make download
make run
make test
make test-race
make lint
make fmt
make build
make site
make site-serve
make screenshots
```

`make run`, `make build`, and frontend-related checks ensure npm dependencies exist before invoking TypeScript. `web/dist` is generated and ignored by Git.

## Development tools

Shared development tools come from [gi8lino/dev-tools](https://github.com/gi8lino/dev-tools).

Lore keeps only the dev-tools bootstrap file in the repository:

```text
bin/dev-tools.mk
```

The Makefile pins the dev-tools release:

```makefile
# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.7.0
```

and includes only the modules it needs:

```makefile
include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,port)
include $(call dev-tools-module,browser)
include $(call dev-tools-module,help)
```

Missing modules and their executables are downloaded automatically from the pinned GitHub release when Make needs them. They are cached below:

```text
bin/.dev-tools/<DEV_TOOLS_VERSION>/
```

Changing `DEV_TOOLS_VERSION` selects a separate versioned cache, so upgrades and rollbacks do not depend on file modification times.

The shared modules provide:

- semantic-version tagging targets such as `make current`, `make patch`, `make minor`, `make major`, and `make push`;
- persistent named development ports through `dev-port`;
- browser startup through `open-browser`;
- generated Make help through `make-help`;
- generic helpers such as `go-install-tool` and `github-release-install`.

Project-specific tools such as `golangci-lint` remain installed in `bin/`. `go-install-tool` maintains their versioned binaries and stable symlinks.

## Local ports

Local startup requires Python 3 because `dev-port` is implemented in Python.

Named ports are stored in the Git-ignored `.dev-ports.json` and reused across Make invocations. Lore uses separate assignments for the application, PostgreSQL, PDF service, and documentation site.

```makefile
LORE_ASSIGNED_PORT ?= $(call dev-port,app)
DB_ASSIGNED_PORT ?= $(call dev-port,postgres)
PDF_ASSIGNED_PORT ?= $(call dev-port,pdf)
SITE_PORT ?= $(call dev-port,site)
```

This supports starting components independently:

```sh
make dev-build
make postgres
make html-pdf
make serve
```

`make run` builds and starts everything together and opens the browser once Lore responds. `make open` can also be run from another terminal.

`make ports` prints the saved application, PostgreSQL, and PDF addresses.

Override and save fixed service ports when necessary:

```sh
make ports \
  LORE_ASSIGNED_PORT=8080 \
  DB_ASSIGNED_PORT=5433 \
  PDF_ASSIGNED_PORT=8081
```

The port helper can also be used directly through the path provided by the dev-tools port module:

```sh
$(make -s --no-print-directory -f - <<'EOF'
include bin/dev-tools.mk
include $(call dev-tools-module,port)

print:
	@echo "$(DEV_PORT)"
EOF
) postgres
```

Normally there is no reason to invoke it directly; the Make targets handle persistent assignments.

Stop services before `make ports-reset`. Saved ports are reused even while occupied and are not reservations: another process can still take a free port between allocation and service startup.

## Documentation site

Lore builds its own documentation using the static-site generator:

```sh
make site
```

The source and site configuration live under:

```text
docs/
├── content/
└── site.toml
```

The generated output is written to `docs/site/`.

To build, serve, and automatically open the documentation locally:

```sh
make site-serve
```

The documentation server receives a persistent dynamic port through the shared `dev-port` module:

```makefile
SITE_PORT ?= $(call dev-port,site)
```

The site URL is therefore stable across Make invocations for the current checkout without requiring a globally fixed development port.

An explicit port can still be selected when useful:

```sh
make site-serve SITE_PORT=8081
```

`open-browser` starts in the background before the local HTTP server and waits until the site responds before opening the default browser.

## Frontend

All authored frontend code is TypeScript under `web/src/ts`; frontend tests are TypeScript under `test/ts`. `tsc` emits native ES modules into `web/dist/js`. The service worker has its own TypeScript project so Web Worker types do not leak into the browser DOM project. Node-based tests likewise have a separate TypeScript configuration.

The production deployment does not need Node.js. Node/npm/TypeScript exist only while building assets.

`make screenshots` starts a disposable PostgreSQL container, imports `docs/content` into a temporary Lore instance, and regenerates the images under `docs/assets/screenshots`. It uses Playwright with the locally installed Chrome channel and requires Docker, Chrome, `curl`, and `zip`. Set `SCREENSHOT_BROWSER_CHANNEL` to another Playwright Chromium channel when needed.

CSS remains framework-free and is split by responsibility under `web/src/css`. `scripts/web/build-css.sh` bundles the CSS entrypoint.

## Backend

Go code is formatted with `gofmt`; tests use the standard testing package plus Testify where appropriate. PostgreSQL queries and migrations live in `internal/store`.

Read [Architecture](architecture.md) for dependency rules.

### PostgreSQL integration tests

Set `LORE_TEST_DATABASE_URL` to a disposable PostgreSQL database URL, then run `go test -race ./internal/store`. These tests create and remove isolated schemas and require schema creation privileges. Without the variable they are skipped. Coverage includes simultaneous application startup, OIDC identity persistence across database reconnections, and saved-search update conflicts with their original database cause.

Startup migrations run together in one transaction under a database advisory lock. Concurrent instances wait for that transaction before checking migration history; a failed migration rolls back the pending batch so a later startup can retry.

### HTTP error contracts

Handlers translate expected service errors into HTTP responses. `httpresponse.InternalServerError` logs the original error and always writes a safe 500 response with a generated reference. The same reference is written as `error_reference` in the server log so an administrator can correlate a browser error with its diagnostic cause. It is the fallback after expected errors are handled; it does not classify errors itself. Application validation belongs in services; request parsing and transport validation belong in handlers. `httpresponse` only serializes the response.

`domain.ValidationError` carries safe field messages and an optional diagnostic cause; the service aliases use the same type. Persistence validation never depends on HTTP. Known SQL constraints are translated by name and code, preserving their causes for `errors.Is` and `errors.As`. Unknown constraints remain infrastructure failures.

The following contracts were checked against the service implementations and their store dependencies. A missing row is not automatically a missing requested resource: settings use defaults, list queries can return empty results, and missing dependencies during rendering are server failures.

| Service / operation          | Expected failures and HTTP handling                                                                                                                                                                                                                                                                                                                                                 |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Catalog                      | Requested page, alias, revision, and permalink lookups can return not found. Missing revisions keep their own identity, and alias infrastructure failures are propagated rather than replaced with a page 404. Page/permalink handlers translate those to 404. Discovery, search, favorites lists, and inventory queries return empty results or infrastructure errors.             |
| Pages                        | Validation → 422; missing page → 404; forbidden group assignment → 403; occupied page/alias path or recycle-bin path → 409; disabled discussions → 403. Same-path moves, moving a tree into itself, and empty comments are validation failures. Missing comments have a distinct error; forbidden assignments identify `group_ids` or `owner_group_id`.                             |
| Drafts                       | Get/save can return not found; save can return forbidden or validation errors. The draft translator handles these. Listing returns empty results for invalid user IDs or limits; deletion is idempotent. Remaining list/delete errors are infrastructure failures; those causes are propagated unchanged.                                                                           |
| Knowledge                    | Saved-search validation → 422, duplicate name → 409, missing owned search → 404. Snippet validation → 422 and missing/duplicate snippets → 404/409 through the admin translator. Graph, snippet lists, and saved-search lists have infrastructure fallbacks. Missing includes/snippets are handled by Markdown expansion.                                                           |
| Notifications                | Inbox reads have an infrastructure fallback. Invalid notification IDs are validation failures; the HTTP adapter rejects malformed path IDs before the service. Marking a missing or foreign notification read is idempotent, while opening a missing or foreign notification returns not found. Opening an item atomically records the read before returning its local destination. |
| Groups                       | Empty names → 422; duplicate names → 409; missing deleted groups or added/removed memberships → 404. Group and member lists return empty results or infrastructure errors. Assignable groups delegate to group lists without a permission error.                                                                                                                                    |
| Templates                    | Empty names → 422; duplicate names → 409; missing templates → 404 for mutations. Lists have an infrastructure fallback. The editor intentionally ignores an unavailable optional template selection.                                                                                                                                                                                |
| Users / identities           | Admin mutations translate not-found, conflict, and forbidden errors. User lookup after membership creation translates not found. Directory, group, and identity lists have infrastructure fallbacks.                                                                                                                                                                                |
| Tokens                       | Both service and persistence layers return typed validation for empty token names. Missing token owners on creation and missing tokens on deletion → 404. Lists and remaining persistence/randomness failures → 500.                                                                                                                                                                |
| Media                        | Empty, oversized, and unsupported uploads → 400/413/415. Deletion translates missing media → 404, ownership denial → 403, and references preventing deletion → 409. Lists and byte-read failures use 500.                                                                                                                                                                           |
| Navigation                   | Missing icon target → 404 through the admin translator. Navigation lists and icons used for rendering have infrastructure fallbacks.                                                                                                                                                                                                                                                |
| Settings / preferences       | Reads return persisted values or defaults. Services validate preference density and sidebar width; request validation also checks settings. Missing selected OIDC groups produce a field validation error. Remaining persistence failures use 500.                                                                                                                                  |
| Administration / recycle bin | Missing mutation targets → 404 through feature translators. Stats, health, audit, and deleted-page lists have infrastructure fallbacks.                                                                                                                                                                                                                                             |
| Sharing / system             | Missing page/share lookup is handled as not found. Token generation, health checks, setup queries, and persistence failures remain server failures.                                                                                                                                                                                                                                 |
| View data / rendering        | Dependencies are settings, preferences, and lists. Failures loading these or rendering templates/Markdown use 500. Exports preserve image lookup errors separately from page errors. Remote PDF generation retains its explicit 502 response.                                                                                                                                       |

Regression tests cover typed input validation before persistence, wrapped expected errors reaching the correct HTTP translators without server-error logging, and safe logged 500 responses for infrastructure failures.

Store input validation also uses the shared type for local credentials and external identity input. Authentication adapters retain their existing public credential and registration responses; malformed provider data and internal failures are not exposed as raw error text. Best-effort audit and mention side effects retain their existing behavior.

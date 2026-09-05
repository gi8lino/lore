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
```

`make run`, `make build`, and frontend-related checks ensure npm dependencies exist before invoking TypeScript. `web/dist` is generated and ignored by Git.

## Frontend

All authored frontend code is TypeScript under `web/src/ts`; frontend tests are TypeScript under `test/ts`. `tsc` emits native ES modules into `web/dist/js`. The service worker has its own TypeScript project so Web Worker types do not leak into the browser DOM project. Node-based tests likewise have a separate TypeScript configuration.

The production deployment does not need Node.js. Node/npm/TypeScript exist only while building assets.

CSS remains framework-free and is split by responsibility under `web/src/css`. `scripts/web/build-css.sh` bundles the CSS entrypoint.

## Backend

Go code is formatted with `gofmt`; tests use the standard testing package plus Testify where appropriate. PostgreSQL queries and migrations live in `internal/store`.

Read [Architecture](architecture.md) for dependency rules.

### PostgreSQL integration tests

Set `LORE_TEST_DATABASE_URL` to a disposable PostgreSQL database URL, then run `go test -race ./internal/store`. These tests create and remove isolated schemas and require schema creation privileges. Without the variable they are skipped. Coverage includes simultaneous application startup and OIDC identity persistence across database reconnections.

Startup migrations run together in one transaction under a database advisory lock. Concurrent instances wait for that transaction before checking migration history; a failed migration rolls back the pending batch so a later startup can retry.

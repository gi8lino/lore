# Architecture

Lore is a layered Go monolith. It runs as one process with one PostgreSQL database, while package dependencies point inward so HTTP and persistence details stay out of application behavior.

The normal request path is:

```text
HTTP -> routes/middleware -> handler -> service -> repository contract -> store -> PostgreSQL
                                      |                                |
                                      +------------> domain <------------+
```

## Main layers

- `cmd` is the minimal process entry point.
- `internal/cli` defines the TinyFlags command tree (`serve`, `build`, `mirror`) and dispatches each command to its owning package.
- `internal/serve` owns `serve` runtime configuration and is the server composition root. It loads assets, opens PostgreSQL, constructs services/authentication/views, and builds the router.
- `internal/routes` registers routes and applies authentication/role policies to already-constructed dependencies.
- `internal/handler`, `internal/middleware`, and `internal/auth` are inbound HTTP adapters.
- `internal/service` owns application use cases and mutation policy.
- `internal/domain` owns persistence-agnostic domain records and shared errors.
- `internal/store` owns SQL, migrations, transactions, and PostgreSQL row mapping.
- focused packages such as `internal/markdown`, `internal/navigation`, `internal/revision`, `internal/pdf`, `internal/icons`, and `internal/httpresponse` provide narrow capabilities.
- `internal/site` owns the `build` command and filesystem/static publishing adapter. It uses the same Markdown/navigation/theme capabilities but does not construct the server, authentication, services, or store.
- `internal/mirror` owns the `mirror` command and exports PostgreSQL-backed content into a deterministic filesystem snapshot without constructing the server.
- `web` and `themes` contain browser assets and theme resources.

## Boundaries

Handlers do not import the concrete store. Services and authenticators declare the persistence capabilities they consume. `internal/serve` is the place where concrete store implementations satisfy those contracts for the running server. SQL and `pgx` stay in `internal/store`.

The static builder and mirror exporter are intentionally outside the server composition path. `lore build` reads files, renders them, and writes static output without opening PostgreSQL; `lore mirror` opens PostgreSQL only to write its deterministic snapshot.

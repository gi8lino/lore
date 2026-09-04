# Docker

The included Compose file starts Lore and PostgreSQL 18.

```sh
docker compose -f deploy/compose.yaml up -d
```

The development stack publishes Lore on `127.0.0.1:8080` and PostgreSQL on `127.0.0.1:5432`. The Lore service receives a PostgreSQL URL through `LORE__DATABASE_URL` and uses `LORE__PUBLIC_URL=http://localhost:8080`.

## Container image

The production image is multi-stage. A Node build stage compiles the TypeScript frontend, a Go build stage embeds the generated web assets into the Lore binary, and the final Alpine image contains the binary plus WeasyPrint and Noto fonts. Node.js and npm are build dependencies only and are not present in the runtime image.

## Persistent data

PostgreSQL owns the persistent wiki data. The Compose deployment uses the `lore-postgres` volume. Lore itself does not require an application data volume for pages or uploads because those are stored in PostgreSQL.

See [Runtime configuration](../configuration/runtime.md) for environment variables.

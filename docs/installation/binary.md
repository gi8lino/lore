# Binary

Lore can run as a normal Go binary when PostgreSQL is reachable.

A development build can be started with:

```sh
make run
```

A release-style local binary can be built with:

```sh
make build
./lore --database-url 'postgres://lore:lore@localhost:5432/lore?sslmode=disable'
```

Run the binary with `--help` to see the deployment-level flags available in the current build.

## Frontend assets

The Go binary embeds `web/dist`. Run `make web` before compiling manually so TypeScript and CSS are emitted. Normal `make build`, `make run`, and the Docker build already do this.

## PDF support

PDF export invokes a local WeasyPrint executable. Install WeasyPrint and suitable fonts when PDF export is required. The official container image installs both.

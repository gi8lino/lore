# Binary

Lore can run as a normal Go binary when PostgreSQL is reachable.

A development build can be started with:

```sh
make run
```

A release-style local binary can be built with:

```sh
make build
./lore serve --database-url 'postgres://lore:lore@localhost:5432/lore?sslmode=disable'
```

Run `./lore --help` to see the available commands or `./lore serve --help` for server runtime flags.

## Frontend assets

The Go binary embeds `web/dist`. Run `make web` before compiling manually so TypeScript and CSS are emitted. Normal `make build`, `make run`, and the Docker build already do this.

## PDF support

The Lore binary does not contain a PDF renderer. Configure a compatible HTML-to-PDF `POST` endpoint in **Administration → Configuration**. A deployment can override the persisted endpoint with `LORE__PDF_URL`.

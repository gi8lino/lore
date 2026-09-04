<p align="center">
  <img src="web/src/lore.svg" alt="Lore" width="480" />
</p>

<p align="center">
  <strong>Keep your knowledge close.</strong><br />
  A small, self-hosted, Markdown-first wiki for documentation.
</p>

Lore gives you a focused place to write, organize, search, and share documentation. It runs as a single Go application backed by PostgreSQL and includes authentication, revision history, search, media, collaboration, and administration.

The same binary can also publish Markdown as a read-only static documentation site without PostgreSQL or a running Lore server.

## Quick start

Start Lore with Docker Compose:

```sh
docker compose -f deploy/compose.yaml up -d
```

By default, Lore is available at:

```text
http://localhost:8080
```

On a fresh database, open Lore and create the first administrator through the setup page.

For production deployments, authentication, Kubernetes, static sites, and other configuration, see the [documentation](https://gi8lino.github.io/lore/).

## Common settings

Environment variables use the `LORE__` prefix.

| Setting                | Default                 | What it changes                                                                |
| ---------------------- | ----------------------- | ------------------------------------------------------------------------------ |
| `LORE__LISTEN_ADDRESS` | `127.0.0.1:8080`        | Address and port Lore listens on.                                              |
| `LORE__PUBLIC_URL`     | `http://localhost:8080` | Externally visible URL of the Lore installation.                               |
| `LORE__DATABASE_URL`   | —                       | PostgreSQL connection URL.                                                     |
| `LORE__LOCAL_LOGIN`    | `false`                 | Enables the local recovery login alongside the configured authentication mode. |

See the [configuration guide](https://gi8lino.github.io/lore/configuration/) for all settings and authentication options.

## Documentation

The full documentation is published at:

**[gi8lino.github.io/lore](https://gi8lino.github.io/lore/)**

## License

Lore is licensed under the [Apache License, Version 2.0](LICENSE).

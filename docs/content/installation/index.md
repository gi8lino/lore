# Installation

Lore has one required runtime dependency: PostgreSQL. PDF export optionally uses a separate HTML-to-PDF service configured in the administration UI or with `LORE__PDF_URL`.

The project ships a Dockerfile and a Docker Compose development deployment. A compiled Lore binary can also run directly when PostgreSQL is available.

- [Docker](docker.md)
- [Binary](binary.md)

{{subpages}}

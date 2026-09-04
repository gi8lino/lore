# Local authentication

Local authentication uses Lore-managed username/password credentials and opaque browser sessions.

The first administrator is created through `/setup` on a fresh database. Passwords must contain at least 12 characters. Password hashes use bcrypt.

A successful sign-in creates a random 32-byte session token. Lore stores only a SHA-256 representation of that token in PostgreSQL and sends the raw value in an HTTP-only cookie. Local sessions expire after 12 hours.

## Recovery login

`LORE__LOCAL_LOGIN=true` can expose `/auth/local` alongside another configured mode as a break-glass login. Administrators can set or replace their local recovery password from the configuration area.

Use local recovery deliberately: it is intended to keep access possible when an external identity provider or proxy is unavailable.

# Local authentication

Local authentication uses Lore-managed username/password credentials and opaque browser sessions.

Administrators can disable an individual local credential from **Administration → Users**. A disabled credential cannot create a new local session, and disabling it revokes its existing local sessions. Other identity bindings and API tokens belonging to the same Lore user are not affected.

The first administrator is created through `/setup` on a fresh database. Passwords must contain at least 12 characters. Password hashes use bcrypt.

A successful sign-in creates a random 32-byte session token. Lore stores only a SHA-256 representation of that token in PostgreSQL and sends the raw value in an HTTP-only cookie. Local sessions expire after 12 hours.

## Recovery login

`LORE__LOCAL_LOGIN=true` can expose `/auth/local` alongside another configured mode as a break-glass login. Administrators can set or replace their local recovery password from the configuration area.

If every application login path is unavailable, an operator with PostgreSQL access can re-enable a local credential directly:

```sql
UPDATE local_credentials
SET enabled = true, updated_at = now()
WHERE user_id = (SELECT id FROM users WHERE username = 'admin');
```

The `/auth/local` endpoint must also be exposed, either by local authentication mode or `LORE__LOCAL_LOGIN=true`.

For database-only emergency recovery when the endpoint is not exposed, enable the credential and temporarily select local mode together:

```sql
BEGIN;
UPDATE local_credentials
SET enabled = true, updated_at = now()
WHERE user_id = (SELECT id FROM users WHERE username = 'admin');
UPDATE application_settings
SET auth_mode = 'local', updated_at = now()
WHERE singleton = true;
COMMIT;
```

After signing in, restore the intended authentication mode in **Administration → Configuration**. A deployment-level `LORE__AUTH_MODE` override takes precedence over these database settings.

Use local recovery deliberately: it is intended to keep access possible when an external identity provider or proxy is unavailable.

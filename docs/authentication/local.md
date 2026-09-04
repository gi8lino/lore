# Local authentication

Local authentication uses Lore-managed username/password credentials and opaque browser sessions.

## Accounts and local credentials

Lore users are not permanently classified as local, OIDC, or trusted-proxy accounts. A single user can have an OIDC identity, authenticate through trusted proxy headers, and also have a local password.

Accounts created automatically by OIDC or trusted-proxy login start without a local password. They cannot use `/auth/local` until an administrator creates their first local password in **Administration → Users**. Users cannot grant themselves this additional login method. After an administrator grants it and the user signs in locally, the user can change it under **Account → Settings** by providing the current password.

The local username is the account's current Lore username. Creating a local password does not expose the local endpoint: `/auth/local` must separately be enabled by selecting local authentication mode or setting `LORE__LOCAL_LOGIN=true`.

While OIDC or trusted-proxy authentication is active, administrators can disable an individual local credential from **Administration → Users**. The control appears only for accounts that have a local password. A disabled credential cannot create a new local session, and disabling it revokes its existing local sessions. OIDC identities, trusted-proxy authentication, and API tokens belonging to the same Lore user are not affected. Lore does not allow this state to be changed from the user editor while local authentication is active, preventing an administrator from disabling the login method currently protecting the installation.

Administrators can also create or replace another account's local password from the same user editor while external authentication is active. Passwords must contain at least 12 characters. Setting a password enables the local credential and revokes its previous local sessions.

The first administrator is created through `/setup` on a fresh database. Passwords must contain at least 12 characters. Password hashes use bcrypt.

A successful sign-in creates a random 32-byte session token. Lore stores only a SHA-256 representation of that token in PostgreSQL and sends the raw value in an HTTP-only cookie. Local sessions expire after 12 hours.

Users signed in with a local password can change their own password from **Account → Settings**. They must provide their current password. Changing it revokes their other local sessions and keeps the current browser signed in with a fresh session.

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

# OIDC

OIDC mode uses authorization-code login with provider discovery. Lore binds an account to the verified `(issuer, subject)` pair; usernames, email addresses, and display names are profile attributes rather than account ownership keys.

## Required deployment secrets

Set:

```text
LORE__OIDC_CLIENT_SECRET
LORE__SESSION_SECRET
```

`LORE__SESSION_SECRET` must contain at least 32 characters. The client secret and session secret are not stored in PostgreSQL.

The issuer, client ID, optional group claim, and group mappings are managed as non-secret application settings.

## Group synchronization

Lore can read a configurable top-level group claim as either a string or string array. Administrators explicitly map external group values to Lore groups. An optional authoritative mode removes mapped Lore memberships when those external values disappear from the current claim.

## Unknown identities

When automatic user creation is disabled, a verified unknown identity is recorded for administrator review. An administrator can approve it as a new user, link it to an existing Lore user, reject it, or reopen a rejected request.

The callback verifies issuer and subject before establishing the Lore session.

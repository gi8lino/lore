# OIDC

OIDC mode uses authorization-code login with provider discovery, S256 PKCE, and an ID-token nonce check. Lore binds an account to the verified `(issuer, subject)` pair; usernames, email addresses, and display names are profile attributes rather than account ownership keys.

## Required deployment secrets

Set:

```text
LORE__OIDC_CLIENT_SECRET
LORE__SESSION_SECRET
```

`LORE__SESSION_SECRET` must contain at least 32 characters. The client secret and session secret are not stored in PostgreSQL.

OIDC browser sessions remain valid across Lore restarts and version updates until they expire, provided `LORE__SESSION_SECRET` and the PostgreSQL database are preserved. Sessions last 12 hours. Explicit session revocation and account disabling still invalidate existing sessions. Changing the session secret invalidates all OIDC sessions and pending logins; keep the same secret on every replica. Login attempts expire after 10 minutes and can complete after a restart with the same configuration. Upgrading from a version without PKCE requires restarting any pending login, but does not invalidate established sessions.

The issuer, client ID, optional group claim, administrator group, and group mappings are managed as non-secret application settings.

## External administrator group

Set **Group claim** to the top-level ID-token claim that contains memberships, then set **Administrator group** to the exact value that should grant Lore administrator access. The claim may be one string or an array of strings. Matching is exact after surrounding whitespace is removed.

Lore evaluates the administrator group during a successful OIDC login and stores the observed result. Membership grants effective administrator access for that OIDC session, but it does not overwrite the account's manually assigned Lore role. A manually assigned administrator therefore remains an administrator after leaving the external group. The user administration page warns when Lore last observed that mismatch so an administrator can deliberately keep or remove the manual role.

Removing a user from the identity-provider group affects new logins. Use **Sign out** in **Administration → Users** to invalidate that user's existing local and OIDC browser sessions immediately.

The identity provider must include the configured claim in the ID token. For Keycloak, add a group-membership or role mapper to the Lore client/client scope and emit its value under the same claim name configured in Lore. No Lore collaboration-group mapping is required solely for administrator elevation.

## Group synchronization

Lore can read a configurable top-level group claim as either a string or string array. Administrators explicitly map external group values to Lore groups. An optional authoritative mode removes mapped Lore memberships when those external values disappear from the current claim.

## Unknown identities

When automatic user creation is disabled, a verified unknown identity is recorded for administrator review. An administrator can approve it as a new user, link it to an existing Lore user, reject it, or reopen a rejected request.

The callback verifies issuer and subject before establishing the Lore session.

Automatically created OIDC accounts do not receive a local password. An administrator may explicitly grant one for `/auth/local`; see [Local authentication](local.md#accounts-and-local-credentials).

## Disabled accounts and sessions

Disabling an account in **Administration → Users** revokes its local and OIDC browser sessions and blocks local login, OIDC login, trusted-proxy authentication, and personal API tokens. Re-enabling it does not restore old sessions; the user must authenticate again.



## Provider and restart requirements

Register the exact callback URL `<LORE__PUBLIC_URL>/auth/callback` with the provider. The client must support authorization-code login with S256 PKCE and return `preferred_username` in the ID token. Lore requires provider discovery to succeed at startup when OIDC is active, so an unavailable provider can prevent startup even when existing session cookies are valid. Preserve the issuer URL, database, client configuration, and deployment secrets when restarting. Lore sessions do not require an in-memory provider token or a refresh token.

# Trusted proxy authentication

Trusted-proxy mode accepts identity information from configured HTTP headers. Lore checks the configured username headers in order and uses the first non-empty value. Email and display name are resolved the same way.

Built-in deployment defaults include common headers such as `X-Forwarded-User`, `X-Auth-Request-User`, and `Remote-User` for usernames.

## Security boundary

Only use this mode when Lore is reachable exclusively through a proxy that removes untrusted client-supplied identity headers and writes its own authenticated headers. If clients can connect directly to Lore or inject those headers, the authentication boundary is broken.

Unknown trusted-proxy identities are subject to Lore's user-registration setting.

Persistent trusted-proxy header lists are managed in **Administration → Configuration**. Deployment-level header flags are primarily useful with the emergency authentication override.

## External administrator group

Configure one or more **Group headers** and an exact **Administrator group** value in **Administration → Configuration**. Lore reads the first non-empty configured group header, splits comma-separated values, and grants effective administrator access when one value matches. This elevation is separate from the manually assigned Lore role: an administrator can still promote or demote the account independently.

Unlike OIDC, trusted-proxy authentication is asserted on every request rather than stored in a Lore login session. Group changes therefore take effect on the next request. The user page records the last assertion and warns when a manually assigned Lore administrator is no longer in the configured external administrator group.

The **Sign out** action revokes local and OIDC browser sessions, but it cannot prevent a trusted proxy from authenticating the user again on the next request. Disable the Lore account when access must be blocked. A disabled account is also denied local login, OIDC login, and personal API-token access.

Automatically created trusted-proxy accounts do not receive a local password. An administrator may explicitly grant one for `/auth/local`; see [Local authentication](local.md#accounts-and-local-credentials).

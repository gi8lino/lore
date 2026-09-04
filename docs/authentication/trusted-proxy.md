# Trusted proxy authentication

Trusted-proxy mode accepts identity information from configured HTTP headers. Lore checks the configured username headers in order and uses the first non-empty value. Email and display name are resolved the same way.

Built-in deployment defaults include common headers such as `X-Forwarded-User`, `X-Auth-Request-User`, and `Remote-User` for usernames.

## Security boundary

Only use this mode when Lore is reachable exclusively through a proxy that removes untrusted client-supplied identity headers and writes its own authenticated headers. If clients can connect directly to Lore or inject those headers, the authentication boundary is broken.

Unknown trusted-proxy identities are subject to Lore's user-registration setting.

Persistent trusted-proxy header lists are managed in **Administration → Configuration**. Deployment-level header flags are primarily useful with the emergency authentication override.

Automatically created trusted-proxy accounts do not receive a local password. An administrator may explicitly grant one for `/auth/local`; see [Local authentication](local.md#accounts-and-local-credentials).

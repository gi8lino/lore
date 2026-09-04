# Authentication

The full Lore server supports `local`, `trusted-proxy`, and `oidc` browser authentication. A `none` mode exists for local recovery/development scenarios. API and media endpoints can additionally accept personal access tokens where their route policy allows it.

Authentication establishes an account; authorization then restricts privileged routes by role. Lore uses three roles: `admin`, `editor`, and `viewer`.

- `admin` can administer Lore and perform destructive page operations.
- `editor` can create and edit content and use editor-only APIs.
- `viewer` can use authenticated read features.

OIDC and trusted-proxy modes can additionally grant effective administrator access from one configured external group. External elevation never replaces the manually assigned Lore role. Account suspension is global: a disabled account cannot authenticate through any browser mode or personal API token.

New-user registration is controlled by an application setting. Unknown OIDC identities can be queued for administrator approval when registration is closed.

Read the mode-specific pages:

- [Local authentication](local.md)
- [Trusted proxy](trusted-proxy.md)
- [OIDC](oidc.md)

Static site mode has no authentication at all; see [Static sites](../static-sites.md).

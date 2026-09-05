# Configuration

Lore separates deployment-level configuration from mutable application settings.

Deployment-level values are supplied as flags or `LORE__` environment variables. They include the listen address, PostgreSQL URL, public URL, recovery authentication overrides, OIDC secrets, theme directory, and logging controls.

Application settings are stored in PostgreSQL and changed through the administration interface. They include browser authentication mode, user registration, discussions, rendering features, trusted-proxy header mappings, and non-secret OIDC settings.

- [Runtime configuration](runtime.md)
- [Authentication](../authentication/index.md)
- [Themes](../themes.md)

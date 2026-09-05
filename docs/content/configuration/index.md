# Configuration

Lore separates deployment-level configuration from mutable application settings.

Deployment-level values are supplied as flags or `LORE__` environment variables. They include the listen address, PostgreSQL URL, public URL, recovery authentication overrides, OIDC secrets, theme directory, and logging controls.

Application settings are stored in PostgreSQL and changed through the administration interface. They include browser authentication mode, user registration, discussions, PDF rendering, rendering features, trusted-proxy header mappings, and non-secret OIDC settings. Deployment-level authentication and PDF values can override the persisted settings; the administration UI shows a warning when an override is active.

The PDF integration includes a service test that renders a fixed two-page diagnostic document. Lore verifies that a PDF was returned, reports its page count and size, and shows the generated document so an administrator can judge the visual result.

- [Runtime configuration](runtime.md)
- [Authentication](../authentication/index.md)
- [Themes](../themes.md)



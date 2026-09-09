# Configuration

Lore separates deployment-level configuration from mutable application settings.

Deployment-level values are supplied as flags or `LORE__` environment variables. They include the listen address, PostgreSQL URL, public URL, recovery authentication overrides, OIDC secrets, theme directory, and logging controls.

Application settings are stored in PostgreSQL and changed through the administration interface. They include browser authentication mode, user registration, discussions, the public `robots.txt` policy, configurable external header links, the default content typography size, PDF rendering, rendering features, trusted-proxy header mappings, and non-secret OIDC settings. Deployment-level authentication values override only the authentication fields they manage. The administration UI marks those fields **Managed by deployment** and makes them read-only while the runtime setting is active. Deployment-level PDF configuration can separately override the persisted renderer endpoint.

The regular server exposes `/robots.txt` without authentication. Administrators can choose **Disallow crawling**, **Allow crawling**, or **Disabled** under **Administration → Configuration**. New installations default to disallowing crawling; disabling the feature makes `/robots.txt` return 404.

External links are rendered beside global search for all authenticated users. Each link has a label and HTTP(S) URL plus optional icon and description. The icon picker combines Lucide interface icons and Simple Icons brand logos; persisted identifiers use the explicit `-lucide` or `-simple` suffix. The description is secondary header text and can carry information such as a version or environment.

The PDF integration supports arbitrary request headers for bearer tokens, API keys, gateways, and other service-specific authentication. Each header can be marked **Sensitive**. Sensitive values are encrypted in PostgreSQL with the deployment-managed `LORE__ENCRYPTION_KEY`, are masked in the normal configuration response, and are returned to the browser only after an administrator explicitly chooses **Reveal**. Lore rejects transport-controlled and renderer-protocol headers such as `Host`, `Content-Length`, `Content-Type`, `Accept`, `Transfer-Encoding`, and `Connection`.

The PDF service test uses the endpoint and headers currently entered in the form, including unsaved replacements. Lore renders a fixed two-page diagnostic document, verifies that a PDF was returned, reports its page count and size, and shows the generated document so an administrator can judge the visual result.

{{subpages}}

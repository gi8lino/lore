# Runtime configuration

Lore uses command-line flags and matching `LORE__` environment variables. The database URL is required for `lore serve`; it is not required for `lore build`.

| Flag                    | Environment                 | Purpose                                                                              |
| ----------------------- | --------------------------- | ------------------------------------------------------------------------------------ |
| `--listen-address`      | `LORE__LISTEN_ADDRESS`      | HTTP listen address; defaults to `127.0.0.1:8080`.                                   |
| `--database-url`        | `LORE__DATABASE_URL`        | PostgreSQL connection URL.                                                           |
| `--public-url`          | `LORE__PUBLIC_URL`          | Externally visible base URL; defaults to `http://localhost:8080`.                    |
| `--pdf-url`             | `LORE__PDF_URL`             | Optional runtime override for the configured HTML-to-PDF render endpoint.            |
| `--local-login`         | `LORE__LOCAL_LOGIN`         | Exposes local recovery login alongside another configured authentication mode.       |
| `--theme-directory`     | `LORE__THEME_DIRECTORY`     | Optional directory of TOML theme files that override or extend embedded themes.      |
| `--auth-mode`           | `LORE__AUTH_MODE`           | Emergency authentication override.                                                   |
| `--oidc-issuer`         | `LORE__OIDC_ISSUER`         | OIDC issuer used with the runtime override.                                          |
| `--oidc-client-id`      | `LORE__OIDC_CLIENT_ID`      | OIDC client ID used with the runtime override.                                       |
| `--oidc-client-secret`  | `LORE__OIDC_CLIENT_SECRET`  | OIDC client secret used when OIDC is enabled.                                        |
| `--oidc-session-secret` | `LORE__OIDC_SESSION_SECRET` | Signs OIDC login state/session cookies; when set it must be at least 32 characters.  |
| `--encryption-key`      | `LORE__ENCRYPTION_KEY`      | Base64-encoded 32-byte key used to encrypt sensitive persisted application settings. |
| `--log-format`          | `LORE__LOG_FORMAT`          | `json` or `text`.                                                                    |
| `--debug`               | `LORE__DEBUG`               | Enables verbose diagnostics.                                                         |
| `--access-log`          | `LORE__ACCESS_LOG`          | Enables HTTP access logging.                                                         |

Trusted-proxy username, email, and display-name header lists also have deployment flags and environment-variable forms. Their built-in defaults cover common reverse-proxy headers.

The `--auth-mode` value is a recovery override, not the normal place to configure browser authentication. When it is set, the administration UI shows the effective authentication mode as **Managed by deployment** and does not allow the persisted mode to be changed. Remove the runtime setting and restart Lore to manage the mode in the UI again.

Runtime authentication settings override only their corresponding fields. With the OIDC runtime override, `--oidc-issuer` / `LORE__OIDC_ISSUER` and `--oidc-client-id` / `LORE__OIDC_CLIENT_ID` are also read-only and marked **Managed by deployment**; OIDC group settings remain database-managed. With the trusted-proxy runtime override, the runtime username, email, and display-name header lists are read-only while group-header and administrator-group settings remain database-managed.

The PDF service is also normally configured in **Administration → Configuration**. `--pdf-url` / `LORE__PDF_URL` overrides that persisted endpoint for deployments that want to manage the integration entirely outside Lore. Persisted PDF request headers still apply when the endpoint is overridden. Sensitive header values require `LORE__ENCRYPTION_KEY`; generate a random key with `openssl rand -base64 32` and keep the same value on every Lore replica and across restarts. Changing or losing the key makes existing encrypted values unreadable. See [Authentication](../authentication/index.md).

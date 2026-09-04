# Runtime configuration

Lore uses command-line flags and matching `LORE__` environment variables. The database URL is required for `lore serve`; it is not required for `lore site build`.

| Flag                   | Environment                | Purpose                                                                             |
| ---------------------- | -------------------------- | ----------------------------------------------------------------------------------- |
| `--listen-address`     | `LORE__LISTEN_ADDRESS`     | HTTP listen address; defaults to `127.0.0.1:8080`.                                  |
| `--database-url`       | `LORE__DATABASE_URL`       | PostgreSQL connection URL.                                                          |
| `--public-url`         | `LORE__PUBLIC_URL`         | Externally visible base URL; defaults to `http://localhost:8080`.                   |
| `--local-login`        | `LORE__LOCAL_LOGIN`        | Exposes local recovery login alongside another configured authentication mode.      |
| `--theme-directory`    | `LORE__THEME_DIRECTORY`    | Optional directory of TOML theme files that override or extend embedded themes.     |
| `--auth-mode`          | `LORE__AUTH_MODE`          | Emergency authentication override.                                                  |
| `--oidc-issuer`        | `LORE__OIDC_ISSUER`        | OIDC issuer used with the runtime override.                                         |
| `--oidc-client-id`     | `LORE__OIDC_CLIENT_ID`     | OIDC client ID used with the runtime override.                                      |
| `--oidc-client-secret` | `LORE__OIDC_CLIENT_SECRET` | OIDC client secret used when OIDC is enabled.                                       |
| `--session-secret`     | `LORE__SESSION_SECRET`     | Signs OIDC login state/session cookies; when set it must be at least 32 characters. |
| `--log-format`         | `LORE__LOG_FORMAT`         | `json` or `text`.                                                                   |
| `--debug`              | `LORE__DEBUG`              | Enables verbose diagnostics.                                                        |
| `--access-log`         | `LORE__ACCESS_LOG`         | Enables HTTP access logging.                                                        |

Trusted-proxy username, email, and display-name header lists also have deployment flags and environment-variable forms. Their built-in defaults cover common reverse-proxy headers.

The `--auth-mode` value is a recovery override, not the normal place to configure browser authentication. Use the administration UI for persistent authentication settings. See [Authentication](../authentication/index.md).

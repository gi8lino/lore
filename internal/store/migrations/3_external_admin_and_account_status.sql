ALTER TABLE users
ADD COLUMN enabled boolean NOT NULL DEFAULT true,
ADD COLUMN session_version bigint NOT NULL DEFAULT 1,
ADD COLUMN oidc_admin_observed boolean NOT NULL DEFAULT false,
ADD COLUMN oidc_external_admin boolean NOT NULL DEFAULT false,
ADD COLUMN trusted_proxy_admin_observed boolean NOT NULL DEFAULT false,
ADD COLUMN trusted_proxy_external_admin boolean NOT NULL DEFAULT false;

ALTER TABLE application_settings
ADD COLUMN oidc_admin_group text NOT NULL DEFAULT '',
ADD COLUMN trusted_group_headers text[] NOT NULL DEFAULT ARRAY['X-Forwarded-Groups', 'X-Auth-Request-Groups']::text[],
ADD COLUMN trusted_admin_group text NOT NULL DEFAULT '';

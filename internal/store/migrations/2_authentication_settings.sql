ALTER TABLE application_settings
  ADD COLUMN auth_mode text NOT NULL DEFAULT 'none' CHECK (auth_mode IN ('none', 'trusted-proxy', 'oidc')),
  ADD COLUMN oidc_issuer text NOT NULL DEFAULT '',
  ADD COLUMN oidc_client_id text NOT NULL DEFAULT '',
  ADD COLUMN trusted_username_headers text[] NOT NULL DEFAULT ARRAY['X-Forwarded-User', 'X-Auth-Request-User', 'Remote-User']::text[],
  ADD COLUMN trusted_email_headers text[] NOT NULL DEFAULT ARRAY['X-Forwarded-Email', 'X-Auth-Request-Email', 'X-Authentik-Email']::text[],
  ADD COLUMN trusted_display_name_headers text[] NOT NULL DEFAULT ARRAY['X-Forwarded-Name', 'X-Auth-Request-Preferred-Username', 'X-Authentik-Name']::text[];

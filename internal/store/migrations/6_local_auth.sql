ALTER TABLE application_settings
  DROP CONSTRAINT IF EXISTS application_settings_auth_mode_check;

ALTER TABLE application_settings
  ADD CONSTRAINT application_settings_auth_mode_check
  CHECK (auth_mode IN ('none', 'local', 'trusted-proxy', 'oidc'));

CREATE TABLE local_credentials (
  user_id bigint PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE local_sessions (
  token_hash text PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX local_sessions_user_idx ON local_sessions (user_id);
CREATE INDEX local_sessions_expiry_idx ON local_sessions (expires_at);

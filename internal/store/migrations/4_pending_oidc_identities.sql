CREATE TABLE pending_oidc_identities (
  id bigserial PRIMARY KEY,
  issuer text NOT NULL,
  subject text NOT NULL,
  username text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  display_name text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'rejected')),
  first_seen_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (issuer, subject)
);

CREATE INDEX pending_oidc_identities_status_idx
  ON pending_oidc_identities (status, last_seen_at DESC);

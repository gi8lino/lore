CREATE TABLE oidc_identities (
  issuer text NOT NULL,
  subject text NOT NULL,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (issuer, subject),
  UNIQUE (user_id, issuer)
);

CREATE INDEX oidc_identities_user_idx ON oidc_identities (user_id);

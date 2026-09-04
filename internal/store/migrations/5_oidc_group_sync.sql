ALTER TABLE application_settings
  ADD COLUMN oidc_group_claim text NOT NULL DEFAULT 'groups',
  ADD COLUMN oidc_group_sync boolean NOT NULL DEFAULT false,
  ADD COLUMN oidc_groups_authoritative boolean NOT NULL DEFAULT true;

CREATE TABLE oidc_group_mappings (
  oidc_group text PRIMARY KEY,
  group_id bigint NOT NULL REFERENCES wiki_groups (id) ON DELETE CASCADE
);

CREATE INDEX oidc_group_mappings_group_idx ON oidc_group_mappings (group_id);

CREATE TABLE page_access_rules (
  id bigserial PRIMARY KEY,
  path text NOT NULL,
  group_id bigint NOT NULL REFERENCES wiki_groups (id) ON DELETE CASCADE,
  access text NOT NULL CHECK (access IN ('view', 'edit')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (path, group_id)
);

CREATE INDEX page_access_rules_path_idx ON page_access_rules (path);

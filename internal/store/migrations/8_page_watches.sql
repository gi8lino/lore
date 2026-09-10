CREATE TABLE page_watches (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  path text NOT NULL,
  scope text NOT NULL CHECK (scope IN ('page', 'subtree')),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, path)
);

CREATE INDEX page_watches_path_idx ON page_watches (path, scope, user_id);

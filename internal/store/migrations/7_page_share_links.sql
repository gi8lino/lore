CREATE TABLE page_share_links (
  id bigserial PRIMARY KEY,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  created_by bigint REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);

CREATE INDEX page_share_links_page_idx
  ON page_share_links (page_id, created_at DESC);

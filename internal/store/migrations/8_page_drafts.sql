CREATE TABLE page_drafts (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  draft_key text NOT NULL,
  page_id bigint REFERENCES pages (id) ON DELETE CASCADE,
  base_revision integer NOT NULL DEFAULT 0,
  title text NOT NULL DEFAULT '',
  slug text NOT NULL DEFAULT '',
  form_values jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, draft_key)
);

CREATE INDEX page_drafts_user_updated_idx
  ON page_drafts (user_id, updated_at DESC);

CREATE INDEX page_drafts_page_idx
  ON page_drafts (page_id)
  WHERE page_id IS NOT NULL;

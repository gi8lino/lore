ALTER TABLE pages
  ADD COLUMN rendered_html text NOT NULL DEFAULT '',
  ADD COLUMN rendered_contents jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN render_fingerprint text NOT NULL DEFAULT '',
  ADD COLUMN rendered_at timestamptz;

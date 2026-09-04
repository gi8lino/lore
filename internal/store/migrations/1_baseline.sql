CREATE TABLE users (
  id bigserial PRIMARY KEY,
  username text NOT NULL UNIQUE,
  email text NOT NULL DEFAULT '',
  display_name text NOT NULL DEFAULT '',
  role text NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin', 'editor', 'viewer')),
  created_at timestamptz NOT NULL DEFAULT now(),
  last_login timestamptz
);

CREATE TABLE wiki_groups (
  id bigserial PRIMARY KEY,
  name text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX wiki_groups_name_ci_idx ON wiki_groups (lower(name));

CREATE TABLE user_groups (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  group_id bigint NOT NULL REFERENCES wiki_groups (id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, group_id)
);

CREATE INDEX user_groups_group_idx ON user_groups (group_id, user_id);

CREATE TABLE application_settings (
  singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
  allow_user_registration boolean NOT NULL DEFAULT true,
  render_wiki_links boolean NOT NULL DEFAULT true,
  render_callouts boolean NOT NULL DEFAULT true,
  render_tabs boolean NOT NULL DEFAULT true,
  render_details boolean NOT NULL DEFAULT true,
  render_tables boolean NOT NULL DEFAULT true,
  render_table_styles boolean NOT NULL DEFAULT true,
  render_table_sorting boolean NOT NULL DEFAULT true,
  render_table_filtering boolean NOT NULL DEFAULT true,
  render_strikethrough boolean NOT NULL DEFAULT true,
  render_task_lists boolean NOT NULL DEFAULT true,
  render_autolinks boolean NOT NULL DEFAULT true,
  render_syntax_highlighting boolean NOT NULL DEFAULT true,
  render_mermaid boolean NOT NULL DEFAULT true,
  render_footnotes boolean NOT NULL DEFAULT false,
  render_definition_lists boolean NOT NULL DEFAULT false,
  render_typographer boolean NOT NULL DEFAULT false,
  render_content_language text NOT NULL DEFAULT 'en',
  render_coding_ligatures boolean NOT NULL DEFAULT false,
  discussions_enabled boolean NOT NULL DEFAULT true,
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO application_settings (singleton) VALUES (true);

CREATE TABLE user_preferences (
  user_id bigint PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  theme text NOT NULL DEFAULT '',
  show_page_contents boolean NOT NULL DEFAULT true,
  navigation_density text NOT NULL DEFAULT 'comfortable' CHECK (navigation_density IN ('comfortable', 'compact')),
  sidebar_width integer NOT NULL DEFAULT 280 CHECK (sidebar_width BETWEEN 220 AND 420),
  show_navigation_guides boolean NOT NULL DEFAULT true,
  remember_navigation_state boolean NOT NULL DEFAULT true,
  show_pinned_pages boolean NOT NULL DEFAULT true,
  show_recently_viewed boolean NOT NULL DEFAULT false,
  show_navigation_page_counts boolean NOT NULL DEFAULT false,
  expanded_navigation text[] NOT NULL DEFAULT '{}',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE pages (
  id bigserial PRIMARY KEY,
  slug text NOT NULL UNIQUE,
  title text NOT NULL,
  markdown_content text NOT NULL DEFAULT '',
  content_language text NOT NULL DEFAULT '',
  created_by bigint REFERENCES users (id),
  updated_by bigint REFERENCES users (id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  view_count bigint NOT NULL DEFAULT 0,
  deleted_at timestamptz,
  deleted_by bigint REFERENCES users (id),
  status text NOT NULL DEFAULT 'verified' CHECK (status IN ('draft', 'verified', 'deprecated', 'archived')),
  owner_group_id bigint REFERENCES wiki_groups (id) ON DELETE SET NULL,
  last_reviewed_at timestamptz,
  review_interval_days integer NOT NULL DEFAULT 0 CHECK (review_interval_days >= 0),
  deprecated_target text NOT NULL DEFAULT '',
  search_vector tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A')
    || setweight(to_tsvector('english', coalesce(markdown_content, '')), 'B')
  ) STORED
);

CREATE INDEX pages_search_idx ON pages USING gin (search_vector);
CREATE INDEX pages_updated_idx ON pages (updated_at DESC);
CREATE INDEX pages_deleted_at_idx ON pages (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX pages_status_idx ON pages (status) WHERE deleted_at IS NULL;
CREATE INDEX pages_review_due_idx ON pages (last_reviewed_at, review_interval_days)
  WHERE deleted_at IS NULL AND review_interval_days > 0;

CREATE TABLE page_revisions (
  id bigserial PRIMARY KEY,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  revision_number integer NOT NULL,
  markdown_content text NOT NULL,
  created_by bigint REFERENCES users (id),
  created_at timestamptz NOT NULL DEFAULT now(),
  message text NOT NULL DEFAULT '',
  UNIQUE (page_id, revision_number)
);

CREATE TABLE tags (
  id bigserial PRIMARY KEY,
  name text NOT NULL UNIQUE
);

CREATE TABLE page_tags (
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  tag_id bigint NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  PRIMARY KEY (page_id, tag_id)
);

CREATE TABLE page_groups (
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  group_id bigint NOT NULL REFERENCES wiki_groups (id) ON DELETE CASCADE,
  PRIMARY KEY (page_id, group_id)
);

CREATE INDEX page_groups_group_idx ON page_groups (group_id, page_id);

CREATE TABLE api_tokens (
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  token_hash text NOT NULL UNIQUE,
  created_by bigint REFERENCES users (id),
  user_id bigint REFERENCES users (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used timestamptz,
  expires_at timestamptz
);

CREATE INDEX api_tokens_user_idx ON api_tokens (user_id, created_at DESC);

CREATE TABLE favorites (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, page_id)
);

CREATE TABLE page_views (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  viewed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, page_id)
);

CREATE TABLE page_links (
  source_page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  target_slug text NOT NULL,
  PRIMARY KEY (source_page_id, target_slug)
);

CREATE INDEX page_links_target_idx ON page_links (target_slug);

CREATE TABLE page_properties (
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  key text NOT NULL,
  value text NOT NULL DEFAULT '',
  PRIMARY KEY (page_id, key)
);

CREATE UNIQUE INDEX page_properties_key_ci_idx ON page_properties (page_id, lower(key));

CREATE TABLE images (
  id bigserial PRIMARY KEY,
  filename text NOT NULL,
  content_type text NOT NULL,
  data bytea NOT NULL,
  size_bytes bigint NOT NULL CHECK (size_bytes > 0),
  uploaded_by bigint REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX images_created_idx ON images (created_at DESC);

CREATE TABLE attachments (
  id bigserial PRIMARY KEY,
  filename text NOT NULL,
  content_type text NOT NULL,
  data bytea NOT NULL,
  size_bytes bigint NOT NULL CHECK (size_bytes > 0),
  uploaded_by bigint REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX attachments_created_idx ON attachments (created_at DESC);

CREATE TABLE navigation_icons (
  path text PRIMARY KEY,
  icon text NOT NULL DEFAULT ''
);

CREATE TABLE page_templates (
  id bigserial PRIMARY KEY,
  name text NOT NULL UNIQUE,
  description text NOT NULL DEFAULT '',
  markdown_content text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX page_templates_name_ci_idx ON page_templates (lower(name));

CREATE TABLE page_aliases (
  alias text PRIMARY KEY,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX page_aliases_page_idx ON page_aliases (page_id);

CREATE TABLE audit_events (
  id bigserial PRIMARY KEY,
  user_id bigint REFERENCES users (id) ON DELETE SET NULL,
  action text NOT NULL,
  object_type text NOT NULL DEFAULT '',
  object_key text NOT NULL DEFAULT '',
  detail text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_created_idx ON audit_events (created_at DESC, id DESC);

CREATE TABLE knowledge_snippets (
  id bigserial PRIMARY KEY,
  kind text NOT NULL CHECK (kind IN ('variable', 'snippet')),
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  content text NOT NULL DEFAULT '',
  updated_by bigint REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (kind, name)
);

CREATE UNIQUE INDEX knowledge_snippets_kind_name_ci_idx ON knowledge_snippets (kind, lower(name));

CREATE TABLE saved_searches (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name text NOT NULL,
  query text NOT NULL,
  pinned boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, name)
);

CREATE UNIQUE INDEX saved_searches_user_name_ci_idx ON saved_searches (user_id, lower(name));

CREATE TABLE notifications (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  kind text NOT NULL DEFAULT 'info',
  title text NOT NULL,
  body text NOT NULL DEFAULT '',
  url text NOT NULL DEFAULT '',
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notifications_user_idx ON notifications (user_id, read_at, created_at DESC);

CREATE TABLE page_comments (
  id bigserial PRIMARY KEY,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  user_id bigint REFERENCES users (id) ON DELETE SET NULL,
  anchor text NOT NULL DEFAULT '',
  body text NOT NULL,
  resolved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX page_comments_page_idx ON page_comments (page_id, resolved_at, created_at);

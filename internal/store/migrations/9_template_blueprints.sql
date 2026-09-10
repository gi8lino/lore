ALTER TABLE page_templates
ADD COLUMN path_prefix text NOT NULL DEFAULT '',
ADD COLUMN icon text NOT NULL DEFAULT '',
ADD COLUMN tags text[] NOT NULL DEFAULT '{}',
ADD COLUMN status text NOT NULL DEFAULT 'verified' CHECK (status IN ('draft', 'verified', 'deprecated', 'archived')),
ADD COLUMN owner_group_id bigint REFERENCES wiki_groups (id) ON DELETE SET NULL,
ADD COLUMN review_interval_days integer NOT NULL DEFAULT 0 CHECK (review_interval_days >= 0),
ADD COLUMN properties jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(properties) = 'object'),
ADD COLUMN fields jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(fields) = 'array');

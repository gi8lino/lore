ALTER TABLE application_settings
ADD COLUMN external_links jsonb NOT NULL DEFAULT '[]'::jsonb
CHECK (jsonb_typeof(external_links) = 'array');

ALTER TABLE application_settings
ADD COLUMN robots_policy text NOT NULL DEFAULT 'disallow'
CHECK (robots_policy IN ('allow', 'disallow', 'none'));

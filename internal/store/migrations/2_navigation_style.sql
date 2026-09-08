ALTER TABLE user_preferences
ADD COLUMN IF NOT EXISTS navigation_style text NOT NULL DEFAULT 'sidebar'
CHECK (navigation_style IN ('sidebar', 'topbar', 'tree'));

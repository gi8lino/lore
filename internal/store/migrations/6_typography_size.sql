ALTER TABLE application_settings
ADD COLUMN default_typography_size text NOT NULL DEFAULT 'compact'
CHECK (default_typography_size IN ('compact', 'standard', 'large'));

ALTER TABLE user_preferences
ADD COLUMN typography_size text NOT NULL DEFAULT ''
CHECK (typography_size IN ('', 'compact', 'standard', 'large'));

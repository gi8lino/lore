ALTER TABLE webhooks
ADD COLUMN body_template text NOT NULL DEFAULT '',
ADD COLUMN retry_enabled boolean NOT NULL DEFAULT false,
ADD COLUMN retry_count integer NOT NULL DEFAULT 2 CHECK (retry_count >= 0),
ADD COLUMN retry_backoff_ms bigint NOT NULL DEFAULT 1000 CHECK (retry_backoff_ms >= 0),
ADD COLUMN retry_max_backoff_ms bigint NOT NULL DEFAULT 30000 CHECK (retry_max_backoff_ms >= 0),
ADD COLUMN retry_jitter boolean NOT NULL DEFAULT true;

ALTER TABLE webhooks
DROP COLUMN secret;

CREATE TABLE webhook_headers (
  id bigserial PRIMARY KEY,
  webhook_id bigint NOT NULL REFERENCES webhooks (id) ON DELETE CASCADE,
  name text NOT NULL,
  value text NOT NULL,
  sensitive boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX webhook_headers_name_ci_idx
ON webhook_headers (webhook_id, lower(name));

ALTER TABLE webhook_deliveries
ADD COLUMN attempts integer NOT NULL DEFAULT 1 CHECK (attempts >= 0);

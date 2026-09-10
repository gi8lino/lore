CREATE TABLE webhooks (
  id bigserial PRIMARY KEY,
  name text NOT NULL UNIQUE,
  url text NOT NULL,
  events text[] NOT NULL DEFAULT '{}',
  secret text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX webhooks_name_ci_idx ON webhooks (lower(name));

CREATE TABLE webhook_deliveries (
  id bigserial PRIMARY KEY,
  webhook_id bigint NOT NULL REFERENCES webhooks (id) ON DELETE CASCADE,
  event text NOT NULL,
  status_code integer NOT NULL DEFAULT 0,
  error text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX webhook_deliveries_recent_idx ON webhook_deliveries (created_at DESC, id DESC);

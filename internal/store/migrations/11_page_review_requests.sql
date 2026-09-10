CREATE TABLE page_review_requests (
  id bigserial PRIMARY KEY,
  page_id bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  revision_number integer NOT NULL,
  requested_by bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  reviewed_by bigint REFERENCES users (id) ON DELETE SET NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'changes_requested', 'approved')),
  note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX page_review_requests_pending_idx
ON page_review_requests (page_id)
WHERE status = 'pending';

CREATE INDEX page_review_requests_page_idx
ON page_review_requests (page_id, created_at DESC, id DESC);

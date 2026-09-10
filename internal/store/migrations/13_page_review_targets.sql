ALTER TABLE page_review_requests
ADD COLUMN reviewer_group_id bigint REFERENCES wiki_groups (id) ON DELETE SET NULL,
ADD COLUMN decision_note text NOT NULL DEFAULT '',
ADD COLUMN previous_status text NOT NULL DEFAULT 'draft';

ALTER TABLE page_review_requests
DROP CONSTRAINT page_review_requests_status_check;

ALTER TABLE page_review_requests
ADD CONSTRAINT page_review_requests_status_check
CHECK (status IN ('pending', 'changes_requested', 'approved', 'canceled', 'superseded'));

CREATE TABLE page_review_request_reviewers (
  request_id bigint NOT NULL REFERENCES page_review_requests (id) ON DELETE CASCADE,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  PRIMARY KEY (request_id, user_id)
);

CREATE INDEX page_review_request_reviewers_user_idx
ON page_review_request_reviewers (user_id, request_id);

UPDATE page_review_requests rr
SET reviewer_group_id = p.owner_group_id
FROM pages p
WHERE p.id = rr.page_id
  AND rr.reviewer_group_id IS NULL
  AND rr.status = 'pending';

UPDATE page_review_requests
SET decision_note = note
WHERE status IN ('approved', 'changes_requested')
  AND decision_note = '';

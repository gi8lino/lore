package store

import (
	"context"
	"errors"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5"
)

// PageReviewRequest returns the newest review request for a page.
func (s *Store) PageReviewRequest(ctx context.Context, slug string) (domain.PageReviewRequest, error) {
	var item domain.PageReviewRequest
	err := s.pool.QueryRow(ctx, `
SELECT rr.id,p.slug,rr.revision_number,rr.requested_by,requester.username,
       coalesce(rr.reviewed_by,0),coalesce(reviewer.username,''),rr.status,rr.note,rr.created_at,rr.updated_at
FROM page_review_requests rr
JOIN pages p ON p.id=rr.page_id
JOIN users requester ON requester.id=rr.requested_by
LEFT JOIN users reviewer ON reviewer.id=rr.reviewed_by
WHERE p.slug=$1 AND p.deleted_at IS NULL
ORDER BY rr.created_at DESC,rr.id DESC
LIMIT 1`, slug).Scan(
		&item.ID, &item.PageSlug, &item.RevisionNumber, &item.RequestedBy, &item.RequestedByName,
		&item.ReviewedBy, &item.ReviewedByName, &item.Status, &item.Note, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PageReviewRequest{}, nil
	}
	return item, err
}

// RequestPageReview opens or refreshes the pending request at the current revision.
func (s *Store) RequestPageReview(ctx context.Context, slug string, actorID int64, note string) (domain.PageReviewRequest, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.PageReviewRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pageID int64
	var ownerGroupID int64
	var revisionNumber int
	err = tx.QueryRow(ctx, `
SELECT p.id,coalesce(p.owner_group_id,0),coalesce(max(r.revision_number),0)
FROM pages p
LEFT JOIN page_revisions r ON r.page_id=p.id
WHERE p.slug=$1 AND p.deleted_at IS NULL
GROUP BY p.id`, slug).Scan(&pageID, &ownerGroupID, &revisionNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PageReviewRequest{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PageReviewRequest{}, err
	}

	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO page_review_requests(page_id,revision_number,requested_by,status,note)
VALUES($1,$2,$3,'pending',$4)
ON CONFLICT(page_id) WHERE status='pending' DO UPDATE
SET revision_number=excluded.revision_number,requested_by=excluded.requested_by,note=excluded.note,
    reviewed_by=NULL,updated_at=now()
RETURNING id`, pageID, revisionNumber, actorID, note).Scan(&id)
	if err != nil {
		return domain.PageReviewRequest{}, mutationError(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE pages SET status='draft',updated_at=now() WHERE id=$1`, pageID); err != nil {
		return domain.PageReviewRequest{}, err
	}

	if ownerGroupID != 0 {
		_, err = tx.Exec(ctx, `
INSERT INTO notifications(user_id,kind,title,body,url)
SELECT DISTINCT u.id,'review','Review requested for ' || p.title,$2,'/pages/' || p.slug
FROM pages p
JOIN user_groups ug ON ug.group_id=p.owner_group_id
JOIN users u ON u.id=ug.user_id AND u.enabled
WHERE p.id=$1 AND u.id<>$3`, pageID, note, actorID)
	} else {
		_, err = tx.Exec(ctx, `
INSERT INTO notifications(user_id,kind,title,body,url)
SELECT u.id,'review','Review requested for ' || p.title,$2,'/pages/' || p.slug
FROM pages p
JOIN users u ON u.role='admin' AND u.enabled
WHERE p.id=$1 AND u.id<>$3`, pageID, note, actorID)
	}
	if err != nil {
		return domain.PageReviewRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PageReviewRequest{}, err
	}
	return s.PageReviewRequest(ctx, slug)
}

// CanReviewPage reports whether the user belongs to the page owner group.
func (s *Store) CanReviewPage(ctx context.Context, slug string, userID int64) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx, `
SELECT CASE
  WHEN p.owner_group_id IS NULL THEN EXISTS(SELECT 1 FROM users WHERE id=$2 AND role='admin')
  ELSE EXISTS(SELECT 1 FROM user_groups WHERE user_id=$2 AND group_id=p.owner_group_id)
END
FROM pages p
WHERE p.slug=$1 AND p.deleted_at IS NULL`, slug, userID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrNotFound
	}
	return allowed, err
}

// DecidePageReview approves a pending request or asks for changes.
func (s *Store) DecidePageReview(ctx context.Context, id int64, expectedSlug string, reviewerID int64, decision, note string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pageID, requesterID int64
	var slug, title string
	var requestedRevision, currentRevision int
	err = tx.QueryRow(ctx, `
SELECT p.id,p.slug,p.title,rr.requested_by,rr.revision_number,
       coalesce((SELECT max(revision_number) FROM page_revisions WHERE page_id=p.id),0)
FROM page_review_requests rr
JOIN pages p ON p.id=rr.page_id
WHERE rr.id=$1 AND p.slug=$2 AND rr.status='pending' AND p.deleted_at IS NULL
FOR UPDATE OF rr`, id, expectedSlug).Scan(&pageID, &slug, &title, &requesterID, &requestedRevision, &currentRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if requestedRevision != currentRevision {
		return "", domain.ErrStaleReview
	}

	tag, err := tx.Exec(ctx, `
UPDATE page_review_requests
SET status=$2,note=$3,reviewed_by=$4,updated_at=now()
WHERE id=$1 AND status='pending'`, id, decision, note, reviewerID)
	if err != nil {
		return "", mutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return "", domain.ErrNotFound
	}
	if decision == "approved" {
		if _, err := tx.Exec(ctx, `UPDATE pages SET status='verified',last_reviewed_at=now(),updated_at=now() WHERE id=$1`, pageID); err != nil {
			return "", err
		}
	}
	body := note
	if body == "" {
		body = "The review was " + decision + "."
	}
	if requesterID != reviewerID {
		if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,kind,title,body,url) VALUES($1,'review',$2,$3,$4)`, requesterID, "Review "+decision+" for "+title, body, "/pages/"+slug); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return slug, nil
}

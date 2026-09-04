package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Attachments returns attachment metadata with Markdown reference counts.
func (s *Store) Attachments(ctx context.Context) ([]Attachment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT a.id,a.filename,a.content_type,a.size_bytes,coalesce(a.uploaded_by,0),coalesce(u.display_name,u.username,''),a.created_at,
       (SELECT count(*) FROM pages p WHERE p.deleted_at IS NULL AND p.markdown_content LIKE '%/attachments/'||a.id::text||'/%')
FROM attachments a
LEFT JOIN users u ON u.id=a.uploaded_by
ORDER BY a.created_at DESC,a.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attachments []Attachment
	for rows.Next() {
		var item Attachment
		if err := rows.Scan(&item.ID, &item.Filename, &item.ContentType, &item.SizeBytes, &item.UploadedBy, &item.Uploader, &item.CreatedAt, &item.UsageCount); err != nil {
			return nil, err
		}
		attachments = append(attachments, item)
	}
	return attachments, rows.Err()
}

// AttachmentInfo returns metadata for one attachment.
func (s *Store) AttachmentInfo(ctx context.Context, id int64) (Attachment, error) {
	var item Attachment
	err := s.pool.QueryRow(ctx, `
SELECT a.id,a.filename,a.content_type,a.size_bytes,coalesce(a.uploaded_by,0),coalesce(u.display_name,u.username,''),a.created_at,
       (SELECT count(*) FROM pages p WHERE p.deleted_at IS NULL AND p.markdown_content LIKE '%/attachments/'||a.id::text||'/%')
FROM attachments a
LEFT JOIN users u ON u.id=a.uploaded_by
WHERE a.id=$1`, id).Scan(
		&item.ID,
		&item.Filename,
		&item.ContentType,
		&item.SizeBytes,
		&item.UploadedBy,
		&item.Uploader,
		&item.CreatedAt,
		&item.UsageCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	return item, err
}

// AttachmentContent returns attachment metadata and bytes.
func (s *Store) AttachmentContent(ctx context.Context, id int64) (AttachmentData, error) {
	var item AttachmentData
	err := s.pool.QueryRow(ctx, `
SELECT id,filename,content_type,size_bytes,coalesce(uploaded_by,0),created_at,data
FROM attachments
WHERE id=$1`, id).
		Scan(&item.ID, &item.Filename, &item.ContentType, &item.SizeBytes, &item.UploadedBy, &item.CreatedAt, &item.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttachmentData{}, ErrNotFound
	}
	return item, err
}

// SaveAttachment stores one uploaded attachment.
func (s *Store) SaveAttachment(ctx context.Context, filename, contentType string, data []byte, userID int64) (Attachment, error) {
	var item Attachment
	err := s.pool.QueryRow(ctx, `
INSERT INTO attachments(filename,content_type,data,size_bytes,uploaded_by)
VALUES($1,$2,$3,$4,$5)
RETURNING id,filename,content_type,size_bytes,coalesce(uploaded_by,0),created_at`, filename, contentType, data, len(data), userID).
		Scan(
			&item.ID,
			&item.Filename,
			&item.ContentType,
			&item.SizeBytes,
			&item.UploadedBy,
			&item.CreatedAt,
		)
	return item, err
}

// DeleteAttachment removes one stored attachment.
func (s *Store) DeleteAttachment(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM attachments
WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

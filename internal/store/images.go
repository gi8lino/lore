package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// SaveImage stores an immutable uploaded image and returns its metadata.
func (s *Store) SaveImage(ctx context.Context, filename, contentType string, data []byte, userID int64) (Image, error) {
	var image Image
	err := s.pool.QueryRow(ctx, `
INSERT INTO images(filename,content_type,data,size_bytes,uploaded_by)
VALUES($1,$2,$3,$4,$5)
RETURNING id,filename,content_type,size_bytes,coalesce(uploaded_by,0),created_at`, filename, contentType, data, int64(len(data)), userID).Scan(
		&image.ID,
		&image.Filename,
		&image.ContentType,
		&image.SizeBytes,
		&image.UploadedBy,
		&image.CreatedAt,
	)
	if err != nil {
		return Image{}, err
	}

	return image, nil
}

// Images returns all uploaded image metadata with exact Markdown reference counts.
func (s *Store) Images(ctx context.Context) ([]Image, error) {
	return s.images(ctx, "")
}

// ImagesByUser returns image metadata uploaded by one user.
func (s *Store) ImagesByUser(ctx context.Context, userID int64) ([]Image, error) {
	return s.images(ctx, `WHERE i.uploaded_by=$1`, userID)
}

// images queries image metadata with an optional WHERE clause and arguments.
func (s *Store) images(ctx context.Context, where string, args ...any) ([]Image, error) {
	rows, err := s.pool.Query(ctx, `
WITH image_references AS (
  SELECT m.captures[1]::bigint AS image_id,count(*) AS usage_count
  FROM pages p
  CROSS JOIN LATERAL regexp_matches(p.markdown_content, '/media/([0-9]+)/', 'g') AS m(captures)
  GROUP BY m.captures[1]::bigint
)
SELECT
  i.id,
  i.filename,
  i.content_type,
  i.size_bytes,
  coalesce(i.uploaded_by,0),
  coalesce(u.display_name,u.username,''),
  i.created_at,
  coalesce(refs.usage_count,0)
FROM images i
LEFT JOIN users u ON u.id=i.uploaded_by
LEFT JOIN image_references refs ON refs.image_id=i.id
`+where+`
ORDER BY i.created_at DESC,i.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		var image Image
		if err := rows.Scan(
			&image.ID,
			&image.Filename,
			&image.ContentType,
			&image.SizeBytes,
			&image.UploadedBy,
			&image.Uploader,
			&image.CreatedAt,
			&image.UsageCount,
		); err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, rows.Err()
}

// ImageInfo returns image metadata with its exact current Markdown reference count.
func (s *Store) ImageInfo(ctx context.Context, id int64) (Image, error) {
	var image Image
	err := s.pool.QueryRow(ctx, `
SELECT
  i.id,
  i.filename,
  i.content_type,
  i.size_bytes,
  coalesce(i.uploaded_by,0),
  coalesce(u.display_name,u.username,''),
  i.created_at,
  coalesce((
    SELECT sum(regexp_count(p.markdown_content, '/media/' || i.id::text || '/'))
    FROM pages p
  ),0)
FROM images i
LEFT JOIN users u ON u.id=i.uploaded_by
WHERE i.id=$1`, id).Scan(
		&image.ID,
		&image.Filename,
		&image.ContentType,
		&image.SizeBytes,
		&image.UploadedBy,
		&image.Uploader,
		&image.CreatedAt,
		&image.UsageCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	return image, err
}

// ImageContent returns the binary payload for one uploaded image.
func (s *Store) ImageContent(ctx context.Context, id int64) (ImageData, error) {
	var image ImageData
	err := s.pool.QueryRow(ctx, `
SELECT filename,content_type,data
FROM images
WHERE id=$1`, id).
		Scan(&image.Filename, &image.ContentType, &image.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImageData{}, ErrNotFound
	}
	return image, err
}

// DeleteImage permanently removes an uploaded image by identifier.
func (s *Store) DeleteImage(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM images
WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

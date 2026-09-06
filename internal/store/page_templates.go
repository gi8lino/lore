package store

import (
	"context"
	"errors"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PageTemplates returns reusable templates in name order.
func (s *Store) PageTemplates(ctx context.Context) ([]domain.PageTemplate, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id,name,description,markdown_content
FROM page_templates
ORDER BY lower(name),id`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var templates []domain.PageTemplate

	for rows.Next() {
		var item domain.PageTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Markdown); err != nil {
			return nil, err
		}

		templates = append(templates, item)
	}

	return templates, rows.Err()
}

// PageTemplate returns one reusable page template.
func (s *Store) PageTemplate(ctx context.Context, id int64) (domain.PageTemplate, error) {
	var item domain.PageTemplate
	err := s.pool.QueryRow(ctx, `
SELECT id,name,description,markdown_content
FROM page_templates
WHERE id=$1`, id).
		Scan(&item.ID, &item.Name, &item.Description, &item.Markdown)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PageTemplate{}, domain.ErrNotFound
	}

	return item, err
}

// CreatePageTemplate creates a reusable page template.
func (s *Store) CreatePageTemplate(ctx context.Context, name, description, markdown string) (domain.PageTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.PageTemplate{}, errors.New("template name is required")
	}

	var item domain.PageTemplate
	err := s.pool.QueryRow(ctx, `
INSERT INTO page_templates(name,description,markdown_content)
VALUES($1,$2,$3)
RETURNING id,name,description,markdown_content`, name, strings.TrimSpace(description), markdown).
		Scan(&item.ID, &item.Name, &item.Description, &item.Markdown)
	if databaseError, ok := errors.AsType[*pgconn.PgError](err); ok && databaseError.Code == "23505" {
		return domain.PageTemplate{}, domain.ErrAlreadyExists
	}

	return item, err
}

// UpdatePageTemplate updates a reusable page template.
func (s *Store) UpdatePageTemplate(ctx context.Context, id int64, name, description, markdown string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("template name is required")
	}

	tag, err := s.pool.Exec(ctx, `
UPDATE page_templates
SET name=$2,description=$3,markdown_content=$4,updated_at=now()
WHERE id=$1`, id, name, strings.TrimSpace(description), markdown)
	if databaseError, ok := errors.AsType[*pgconn.PgError](err); ok && databaseError.Code == "23505" {
		return domain.ErrAlreadyExists
	}
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return err
}

// DeletePageTemplate removes a reusable page template.
func (s *Store) DeletePageTemplate(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM page_templates
WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return err
}

package service

import (
	"context"
)

// templateRepository contains reusable page template operations.
type templateRepository interface {
	PageTemplates(context.Context) ([]PageTemplate, error)
	PageTemplate(context.Context, int64) (PageTemplate, error)
	CreatePageTemplate(context.Context, string, string, string) (PageTemplate, error)
	UpdatePageTemplate(context.Context, int64, string, string, string) error
	DeletePageTemplate(context.Context, int64) error
}

// Templates exposes reusable page template use cases.
type Templates struct{ repository templateRepository }

// NewTemplates constructs the reusable page template service.
func NewTemplates(repository templateRepository) *Templates {
	return &Templates{repository: repository}
}

// PageTemplates returns all reusable page templates.
func (s *Templates) PageTemplates(ctx context.Context) ([]PageTemplate, error) {
	return s.repository.PageTemplates(ctx)
}

// PageTemplate returns a reusable page template by identifier.
func (s *Templates) PageTemplate(ctx context.Context, id int64) (PageTemplate, error) {
	return s.repository.PageTemplate(ctx, id)
}

// CreatePageTemplate creates a reusable page template.
func (s *Templates) CreatePageTemplate(
	ctx context.Context,
	name, description, markdown string,
) (PageTemplate, error) {
	return s.repository.CreatePageTemplate(ctx, name, description, markdown)
}

// UpdatePageTemplate replaces a reusable page template.
func (s *Templates) UpdatePageTemplate(
	ctx context.Context,
	id int64,
	name, description, markdown string,
) error {
	return s.repository.UpdatePageTemplate(ctx, id, name, description, markdown)
}

// DeletePageTemplate removes a reusable page template.
func (s *Templates) DeletePageTemplate(ctx context.Context, id int64) error {
	return s.repository.DeletePageTemplate(ctx, id)
}

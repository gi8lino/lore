package service

import (
	"context"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/icons"
	md "github.com/gi8lino/lore/internal/markdown"
)

// PageTemplateInput contains transport-independent page-blueprint settings.
type PageTemplateInput struct {
	Name               string
	Description        string
	Markdown           string
	PathPrefix         string
	Icon               string
	Tags               []string
	Status             string
	OwnerGroupID       int64
	ReviewIntervalDays int
	Properties         map[string]string
	Fields             []domain.PageTemplateField
}

// templateRepository contains reusable page template operations.
type templateRepository interface {
	PageTemplates(context.Context) ([]domain.PageTemplate, error)
	PageTemplate(context.Context, int64) (domain.PageTemplate, error)
	CreatePageTemplate(context.Context, domain.PageTemplate) (domain.PageTemplate, error)
	UpdatePageTemplate(context.Context, int64, domain.PageTemplate) error
	DeletePageTemplate(context.Context, int64) error
}

// Templates exposes reusable page-blueprint use cases.
type Templates struct{ repository templateRepository }

// NewTemplates constructs the reusable page template service.
func NewTemplates(repository templateRepository) *Templates {
	return &Templates{repository: repository}
}

// PageTemplates returns all reusable page blueprints.
func (s *Templates) PageTemplates(ctx context.Context) ([]domain.PageTemplate, error) {
	return s.repository.PageTemplates(ctx)
}

// PageTemplate returns a reusable page blueprint by identifier.
func (s *Templates) PageTemplate(ctx context.Context, id int64) (domain.PageTemplate, error) {
	return s.repository.PageTemplate(ctx, id)
}

// CreatePageTemplate creates a reusable page blueprint.
func (s *Templates) CreatePageTemplate(ctx context.Context, input PageTemplateInput) (domain.PageTemplate, error) {
	item, err := validatePageTemplate(input)
	if err != nil {
		return domain.PageTemplate{}, err
	}
	return s.repository.CreatePageTemplate(ctx, item)
}

// UpdatePageTemplate replaces a reusable page blueprint.
func (s *Templates) UpdatePageTemplate(ctx context.Context, id int64, input PageTemplateInput) error {
	item, err := validatePageTemplate(input)
	if err != nil {
		return err
	}
	return s.repository.UpdatePageTemplate(ctx, id, item)
}

// DeletePageTemplate removes a reusable page template.
func (s *Templates) DeletePageTemplate(ctx context.Context, id int64) error {
	return s.repository.DeletePageTemplate(ctx, id)
}

func validatePageTemplate(input PageTemplateInput) (domain.PageTemplate, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.PathPrefix = md.Slug(input.PathPrefix)
	input.Icon = strings.TrimSpace(input.Icon)
	if input.Status == "" {
		input.Status = "verified"
	}

	validation := &ValidationError{}
	if input.Name == "" {
		validation.Fields = append(validation.Fields, FieldError{Field: "name", Message: "A template name is required."})
	}
	if !icons.IsIcon(input.Icon) {
		validation.Fields = append(validation.Fields, FieldError{Field: "icon", Message: "Choose an icon from the available icon catalog."})
	}
	if !domain.ValidPageStatus(input.Status) {
		validation.Fields = append(validation.Fields, FieldError{Field: "status", Message: "Choose a valid default page status."})
	}
	if input.OwnerGroupID < 0 || !domain.ValidReviewIntervalDays(input.ReviewIntervalDays) {
		validation.Fields = append(validation.Fields, FieldError{Field: "review_interval_days", Message: "Choose valid review defaults."})
	}

	seenFields := map[string]bool{}
	fields := make([]domain.PageTemplateField, 0, len(input.Fields))
	for _, field := range input.Fields {
		field.Name = strings.TrimSpace(field.Name)
		field.Label = strings.TrimSpace(field.Label)
		field.Default = strings.TrimSpace(field.Default)
		if field.Name == "" && field.Label == "" {
			continue
		}
		if !validTemplateFieldName(field.Name) {
			validation.Fields = append(validation.Fields, FieldError{Field: "fields", Message: "Field names may contain letters, numbers, underscores, and hyphens."})
			continue
		}
		key := strings.ToLower(field.Name)
		if seenFields[key] {
			validation.Fields = append(validation.Fields, FieldError{Field: "fields", Message: "Template field names must be unique."})
			continue
		}
		seenFields[key] = true
		if field.Label == "" {
			field.Label = field.Name
		}
		fields = append(fields, field)
	}

	if len(validation.Fields) > 0 {
		return domain.PageTemplate{}, validation
	}

	properties := map[string]string{}
	for key, value := range input.Properties {
		key = strings.TrimSpace(key)
		if key != "" {
			properties[key] = strings.TrimSpace(value)
		}
	}

	return domain.PageTemplate{
		Name:               input.Name,
		Description:        input.Description,
		Markdown:           input.Markdown,
		PathPrefix:         input.PathPrefix,
		Icon:               input.Icon,
		Tags:               normalizeTags(input.Tags),
		Status:             input.Status,
		OwnerGroupID:       input.OwnerGroupID,
		ReviewIntervalDays: input.ReviewIntervalDays,
		Properties:         properties,
		Fields:             fields,
	}, nil
}

func validTemplateFieldName(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizeTags(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

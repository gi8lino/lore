package service

import "github.com/gi8lino/lore/internal/domain"

// FieldError is shared with repositories so field information survives every layer.
type FieldError = domain.FieldError

// ValidationError is the common application and repository validation contract.
type ValidationError = domain.ValidationError

func newValidationError(field, message string) error {
	return domain.NewValidationError(field, message)
}

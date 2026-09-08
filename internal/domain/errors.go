package domain

import "fmt"

// FieldError describes a safe, actionable input failure.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError carries input failures across persistence and application boundaries.
// Cause is optional diagnostic context and must not be included in HTTP responses.
type ValidationError struct {
	Fields []FieldError
	Cause  error
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Fields[0].Message, e.Cause)
	}
	return e.Fields[0].Message
}

// Unwrap preserves a persistence cause for errors.Is and errors.As.
func (e *ValidationError) Unwrap() error { return e.Cause }

// NewValidationError creates a single-field input failure.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Fields: []FieldError{{Field: field, Message: message}}}
}

// GroupAssignmentError identifies which page group selection is not assignable.
// It deliberately does not distinguish a hidden group from a nonexistent group.
type GroupAssignmentError struct{ Field string }

func (e *GroupAssignmentError) Error() string { return "page group assignment is forbidden" }
func (e *GroupAssignmentError) Unwrap() error { return ErrForbidden }

// Specific missing-resource errors preserve compatibility with ErrNotFound.
var (
	ErrRevisionNotFound = fmt.Errorf("revision: %w", ErrNotFound)
	ErrCommentNotFound  = fmt.Errorf("comment: %w", ErrNotFound)
)

package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
)

// userMessageError marks an error message as safe for direct presentation to a user.
type userMessageError interface {
	error
	UserMessage() string
}

// requestError carries transport-validation details that are safe to return to the caller.
type requestError struct {
	field   string
	message string
	cause   error
}

// newRequestError creates a transport-validation error with an optional field name.
func newRequestError(field, message string, cause error) error {
	return &requestError{
		field:   field,
		message: message,
		cause:   cause,
	}
}

// Error returns the internal diagnostic error text.
func (e *requestError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}

	return "invalid request"
}

// Unwrap preserves the underlying diagnostic cause.
func (e *requestError) Unwrap() error {
	return e.cause
}

// UserMessage returns the message explicitly approved for presentation to a user.
func (e *requestError) UserMessage() string {
	return e.message
}

// userErrorMessage returns an explicitly safe user-facing error message.
func userErrorMessage(err error) (string, bool) {
	userErr, ok := errors.AsType[userMessageError](err)
	if !ok {
		return "", false
	}

	message := strings.TrimSpace(userErr.UserMessage())
	return message, message != ""
}

// writeRequestProblem maps a typed transport-validation error to an HTTP problem.
func writeRequestProblem(w http.ResponseWriter, status int, title, defaultField string, err error) bool {
	requestErr, ok := errors.AsType[*requestError](err)
	if !ok {
		return false
	}

	message, ok := userErrorMessage(requestErr)
	if !ok {
		return false
	}

	field := requestErr.field
	if field == "" {
		field = defaultField
	}

	if field == "" {
		httpresponse.Problem(w, status, title)
		return true
	}

	httpresponse.Problem(w, 
		status,
		title,
		httpresponse.NewFieldProblem(field, message),
	)
	return true
}

// writeValidationProblem maps application validation failures to field problems.
func writeValidationProblem(w http.ResponseWriter, err error, title string) bool {
	validation, ok := errors.AsType[*domain.ValidationError](err)
	if !ok {
		return false
	}

	problems := make([]httpresponse.FieldProblem, 0, len(validation.Fields))

	for _, field := range validation.Fields {
		problems = append(problems, httpresponse.NewFieldProblem(field.Field, field.Message))
	}

	httpresponse.Problem(w, http.StatusUnprocessableEntity, title, problems...)
	return true
}

// writeInternalServerError logs a failure and writes a safe HTTP 500 response.
func writeInternalServerError(logger *slog.Logger, w http.ResponseWriter, err error) {
	logger.Error("request failed", "event", "request_failed", "error", err)
	httpresponse.Problem(w,
		http.StatusInternalServerError,
		"The request could not be processed.")
}

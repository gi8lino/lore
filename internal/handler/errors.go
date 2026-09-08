package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
)

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

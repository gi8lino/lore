package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestWriteMediaUploadProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "empty", err: service.ErrEmptyFile, status: http.StatusBadRequest},
		{name: "too large", err: service.ErrFileTooLarge, status: http.StatusRequestEntityTooLarge},
		{name: "unsupported", err: service.ErrUnsupportedFileType, status: http.StatusUnsupportedMediaType},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			writeMediaUploadProblem(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				response,
				test.err,
				attachmentMedia,
			)

			assert.Equal(t, test.status, response.Code)
		})
	}
}

func TestWriteMediaDeleteProblem(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	writeMediaDeleteProblem(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		response,
		&service.MediaInUseError{References: 2},
		imageMedia,
	)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), "Image is still referenced 2 time(s).")
}

func TestErrorTranslatorsUseProblemResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		write  func(*slog.Logger, http.ResponseWriter, error)
	}{
		{
			name:   "untranslated not found",
			err:    domain.ErrNotFound,
			status: http.StatusInternalServerError,
			write:  writeInternalServerError,
		},
		{
			name:   "page not found",
			err:    domain.ErrNotFound,
			status: http.StatusNotFound,
			write:  writePageProblem,
		},
		{
			name:   "page in bin",
			err:    domain.ErrPageInBin,
			status: http.StatusConflict,
			write:  writePageProblem,
		},
		{
			name:   "discussions disabled",
			err:    service.ErrDiscussionsDisabled,
			status: http.StatusForbidden,
			write:  writePageProblem,
		},
		{
			name: "page validation",
			err: &service.ValidationError{Fields: []service.FieldError{{
				Field:   "slug",
				Message: "A page path is required.",
			}}},
			status: http.StatusUnprocessableEntity,
			write:  writePageProblem,
		},
		{
			name:   "page assignment forbidden",
			err:    domain.ErrForbidden,
			status: http.StatusForbidden,
			write:  writePageSaveProblem,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()

			test.write(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				response,
				test.err,
			)

			assert.Equal(t, test.status, response.Code)
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			assert.True(t, json.Valid(response.Body.Bytes()))
			assert.Contains(t, response.Body.String(), `"error"`)
		})
	}
}

func TestWriteAdminProblem(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()

	writeAdminProblem(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		response,
		domain.ErrAlreadyExists,
		"Group",
	)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Contains(t, response.Body.String(), "Group already exists.")
}

func TestAdminValidationUsesFieldProblems(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	writeAdminProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response,
		&service.ValidationError{Fields: []service.FieldError{{Field: "name", Message: "A group name is required."}}}, "Group")
	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.JSONEq(t, `{"error":"Group validation failed.","problems":{"name":"A group name is required."}}`, response.Body.String())
}

func TestPageProblemsPreserveResourceAndFieldContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err     error
		status  int
		message string
	}{
		{domain.ErrRevisionNotFound, http.StatusNotFound, "Revision not found."},
		{domain.ErrCommentNotFound, http.StatusNotFound, "Comment not found."},
		{&domain.GroupAssignmentError{Field: "owner_group_id"}, http.StatusForbidden, `"owner_group_id"`},
		{&domain.GroupAssignmentError{Field: "group_ids"}, http.StatusForbidden, `"group_ids"`},
	}
	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			writePageSaveProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, fmt.Errorf("operation: %w", test.err))
			assert.Equal(t, test.status, response.Code)
			assert.Contains(t, response.Body.String(), test.message)
		})
	}
}

func TestValidationResponseDoesNotExposeCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("private SQL and connection details")
	err := domain.NewValidationError("group_ids", "Choose an existing group.")
	err.Cause = cause
	response := httptest.NewRecorder()
	assert.True(t, writeValidationProblem(response, fmt.Errorf("update: %w", err), "Validation failed."))
	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "Choose an existing group.")
	assert.NotContains(t, response.Body.String(), cause.Error())
	assert.ErrorIs(t, err, cause)
}

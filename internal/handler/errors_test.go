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

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		writeMediaUploadProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			service.ErrEmptyFile,
			attachmentMedia,
		)

		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("too large", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		writeMediaUploadProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			service.ErrFileTooLarge,
			attachmentMedia,
		)

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	})

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		writeMediaUploadProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			service.ErrUnsupportedFileType,
			attachmentMedia,
		)

		assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
	})
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

	t.Run("untranslated not found", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writeInternalServerError(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			domain.ErrNotFound,
		)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})

	t.Run("page not found", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writePageProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			domain.ErrNotFound,
		)

		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})

	t.Run("page in bin", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writePageProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			domain.ErrPageInBin,
		)

		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})

	t.Run("discussions disabled", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writePageProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			service.ErrDiscussionsDisabled,
		)

		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})

	t.Run("page validation", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writePageProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			&service.ValidationError{Fields: []service.FieldError{{
				Field:   "slug",
				Message: "A page path is required.",
			}}},
		)

		assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})

	t.Run("page assignment forbidden", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		writePageSaveProblem(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			response,
			domain.ErrForbidden,
		)

		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.True(t, json.Valid(response.Body.Bytes()))
		assert.Contains(t, response.Body.String(), `"error"`)
	})
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

	t.Run("Revision not found.", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writePageSaveProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, fmt.Errorf("operation: %w", domain.ErrRevisionNotFound))
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "Revision not found.")
	})

	t.Run("Comment not found.", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writePageSaveProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, fmt.Errorf("operation: %w", domain.ErrCommentNotFound))
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "Comment not found.")
	})

	t.Run(`"owner_group_id"`, func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writePageSaveProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, fmt.Errorf("operation: %w", &domain.GroupAssignmentError{Field: "owner_group_id"}))
		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Contains(t, response.Body.String(), `"owner_group_id"`)
	})

	t.Run(`"group_ids"`, func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writePageSaveProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, fmt.Errorf("operation: %w", &domain.GroupAssignmentError{Field: "group_ids"}))
		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Contains(t, response.Body.String(), `"group_ids"`)
	})
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

package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/model"
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
			handled := writeMediaUploadProblem(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				response,
				test.err,
				attachmentMedia,
			)

			assert.True(t, handled)
			assert.Equal(t, test.status, response.Code)
		})
	}
}

func TestWriteMediaDeleteProblem(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	handled := writeMediaDeleteProblem(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		response,
		&service.MediaInUseError{References: 2},
		imageMedia,
	)

	assert.True(t, handled)
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
			err:    model.ErrNotFound,
			status: http.StatusInternalServerError,
			write:  writeUnexpectedProblem,
		},
		{
			name:   "page not found",
			err:    model.ErrNotFound,
			status: http.StatusNotFound,
			write:  writePageProblem,
		},
		{
			name:   "page in bin",
			err:    model.ErrPageInBin,
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
			err:    model.ErrForbidden,
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
		model.ErrAlreadyExists,
		"Group",
	)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Contains(t, response.Body.String(), "Group already exists.")
}

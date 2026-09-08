package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestExtractedTranslatorsPreserveResponsesAndLogOnlyInternalFailures(t *testing.T) {
	t.Parallel()
	validation := domain.NewValidationError("name", "Required.")
	tests := []struct {
		name    string
		write   func(*slog.Logger, http.ResponseWriter, error)
		err     error
		status  int
		message string
	}{
		{"media", writeMediaReadProblem, domain.ErrNotFound, 404, "Not found."},
		{"permalink", writePermalinkProblem, domain.ErrNotFound, 404, "Not found."},
		{"token owner", writeTokenCreateProblem, domain.ErrNotFound, 404, "User not found."},
		{"token validation", writeTokenCreateProblem, validation, 422, "Required."},
		{"token delete", writeTokenDeleteProblem, domain.ErrNotFound, 404, "Token not found."},
		{"password", writePasswordChangeProblem, auth.ErrInvalidCredentials, 401, "The current password is incorrect."},
		{"preferences", writePreferencesProblem, validation, 422, "Required."},
		{"share", writePublicShareError, domain.ErrNotFound, 404, "Share link not found or no longer available."},
		{"setup conflict", func(l *slog.Logger, w http.ResponseWriter, err error) { writeSetupProblem(&Views{logger: l}, w, err) }, domain.ErrAlreadyExists, 404, "Not found."},
		{"setup forbidden", func(l *slog.Logger, w http.ResponseWriter, err error) { writeSetupProblem(&Views{logger: l}, w, err) }, domain.ErrForbidden, 404, "Not found."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			response := httptest.NewRecorder()
			test.write(logger, response, fmt.Errorf("wrapped: %w", test.err))
			assert.Equal(t, test.status, response.Code)
			assert.Contains(t, response.Body.String(), test.message)
			assert.Empty(t, logs.String())
			failure := errors.New("private persistence details")
			response = httptest.NewRecorder()
			test.write(logger, response, failure)
			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assert.NotContains(t, response.Body.String(), failure.Error())
			assert.Contains(t, logs.String(), failure.Error())
		})
	}
}

type mediaReadFailureStub struct {
	imageService
	attachmentService
	err error
}

func (s mediaReadFailureStub) ImageContent(context.Context, int64) (domain.ImageData, error) {
	return domain.ImageData{}, s.err
}
func (s mediaReadFailureStub) AttachmentContent(context.Context, int64) (domain.AttachmentData, error) {
	return domain.AttachmentData{}, s.err
}

func TestMediaDownloadsUseLoggedTranslator(t *testing.T) {
	t.Parallel()
	for _, image := range []bool{true, false} {
		t.Run(fmt.Sprint(image), func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			stub := mediaReadFailureStub{err: errors.New("read failed")}
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.SetPathValue("id", "1")
			response := httptest.NewRecorder()
			if image {
				ServeImage(stub, logger)(response, request)
			} else {
				ServeAttachment(stub, logger)(response, request)
			}
			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assert.Contains(t, logs.String(), "read failed")
		})
	}
}

func TestLocalLoginTranslationPreservesHTML(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	views := &Views{logger: slog.New(slog.NewTextHandler(&logs, nil)), templates: map[string]*template.Template{
		"login": template.Must(template.New("public-layout").Parse(`{{.AuthError}} {{.AuthNext}}`)),
	}}
	response := httptest.NewRecorder()
	writeLocalLoginProblem(views, response, fmt.Errorf("login: %w", auth.ErrInvalidCredentials), "/pages/home")
	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Contains(t, response.Body.String(), "Invalid username or password.")
	assert.Contains(t, response.Body.String(), "/pages/home")
	assert.Empty(t, logs.String())
}

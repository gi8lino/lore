package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pdfSettingsStub struct {
	settingsService
	url     string
	actorID int64
}

func (s *pdfSettingsStub) SavePDFSettings(_ context.Context, pdfURL string, actorID int64) error {
	s.url = pdfURL
	s.actorID = actorID

	return nil
}

func TestEffectivePDFURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "http://runtime/render", effectivePDFURL("http://runtime/render", "http://saved/render"))
	assert.Equal(t, "http://saved/render", effectivePDFURL("", "http://saved/render"))
	assert.Empty(t, effectivePDFURL("", ""))
}

func TestSaveAdminPDFSettings(t *testing.T) {
	t.Parallel()

	settings := &pdfSettingsStub{}
	form := url.Values{"pdf_url": {" http://html2pdf:8080/render "}}
	request := httptest.NewRequest(http.MethodPost, "/admin/pdf", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	request = auth.WithUser(request, model.User{ID: 7, Role: "admin"})
	response := httptest.NewRecorder()

	SaveAdminPDFSettings(settings, slog.Default())(response, request)

	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, "/admin/configuration#pdf-rendering", response.Header().Get("Location"))
	assert.Equal(t, "http://html2pdf:8080/render", settings.url)
	assert.Equal(t, int64(7), settings.actorID)
}

func TestTestAdminPDFService(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/render", r.URL.Path)

		body, err := io.ReadAll(r.Body)

		require.NoError(t, err)
		assert.Contains(t, string(body), "PDF service test")
		w.Header().Set("Content-Type", "application/pdf")

		_, _ = io.WriteString(w, "%PDF-1.7\nfixture\n")
	}))
	defer server.Close()

	form := url.Values{"pdf_url": {server.URL + "/render"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/pdf/test", strings.NewReader(form.Encode()))

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := httptest.NewRecorder()

	TestAdminPDFService(slog.Default())(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"message":"PDF service rendered a test document successfully."}`, response.Body.String())
}

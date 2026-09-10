package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type variableExportStub struct {
	settingsService
	navigationService
	knowledgeContentStub
	application     domain.ApplicationSettings
	navigationPages []domain.Page
	pdfHeaders      []domain.PDFHeader
}

func (s variableExportStub) ApplicationSettings(context.Context) (domain.ApplicationSettings, error) {
	return s.application, nil
}
func (s variableExportStub) PDFRequestHeaders(context.Context) ([]domain.PDFHeader, error) {
	return s.pdfHeaders, nil
}
func (s variableExportStub) NavigationPages(context.Context) ([]domain.Page, error) {
	return s.navigationPages, nil
}
func (s variableExportStub) Search(_ context.Context, _ string, limit int) ([]domain.Page, error) {
	pages := make([]domain.Page, 0, len(s.pages))
	for _, page := range s.pages {
		pages = append(pages, page)
	}
	slices.SortFunc(pages, func(left, right domain.Page) int {
		return strings.Compare(left.Slug, right.Slug)
	})
	if limit > 0 && len(pages) > limit {
		pages = pages[:limit]
	}
	return pages, nil
}
func (variableExportStub) NavigationIcons(context.Context) (map[string]string, error) {
	return nil, nil
}

func variableExportFixture(t *testing.T) (variableExportStub, *Views, *slog.Logger) {
	t.Helper()
	content := variableTestContent()
	content.pages["guide"] = domain.Page{Slug: "guide", Title: "Deployment", Language: "en", Markdown: "production {{var:environment}}\n\n```text\n{{var:environment}}\n```"}
	stub := variableExportStub{knowledgeContentStub: content}
	stub.application.Rendering = domain.RenderingSettings{Tables: true, WikiLinks: true}
	return stub, &Views{}, slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestReadExportVariables(t *testing.T) {
	t.Parallel()
	t.Run("reads POST values including an empty string", func(t *testing.T) {
		t.Parallel()
		values, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"variables":{"environment":""}}`)))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"environment": ""}, values)
	})
	t.Run("GET ignores URL and body overrides", func(t *testing.T) {
		t.Parallel()
		values, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/?environment=staging", strings.NewReader(`{"variables":{"environment":"staging"}}`)))
		require.NoError(t, err)
		assert.Nil(t, values)
	})
	t.Run("rejects a non-string value", func(t *testing.T) {
		t.Parallel()
		_, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"variables":{"environment":7}}`)))
		assert.Error(t, err)
	})
	t.Run("rejects unknown top-level fields", func(t *testing.T) {
		t.Parallel()
		_, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"variables":{},"save":true}`)))
		assert.Error(t, err)
	})
	t.Run("rejects a null request", func(t *testing.T) {
		t.Parallel()
		_, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`null`)))
		assert.Error(t, err)
	})
	t.Run("rejects trailing JSON values", func(t *testing.T) {
		t.Parallel()
		_, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{} {}`)))
		assert.Error(t, err)
	})
	t.Run("rejects an oversized request body", func(t *testing.T) {
		t.Parallel()
		_, err := readExportVariables(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"variables":{}}`+strings.Repeat(" ", 512<<10))))
		assert.Error(t, err)
	})
}

func TestPreviewPageExportVariables(t *testing.T) {
	t.Parallel()
	t.Run("returns a private clean preview without needing a PDF service", func(t *testing.T) {
		t.Parallel()
		stub, _, logger := variableExportFixture(t)
		request := httptest.NewRequest(http.MethodPost, "/export/preview/guide", strings.NewReader(`{"variables":{"environment":"staging"}}`))
		request.SetPathValue("slug", "guide")
		response := httptest.NewRecorder()
		PreviewPageExport(stub, stub, stub, stub, &exportMediaStub{}, emptyContractServices{}, md.New(), logger)(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
		var result exportPreviewResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		assert.Contains(t, result.Document, "production staging")
		assert.Contains(t, result.Document, "{{var:environment}}")
		assert.NotContains(t, result.Document, "data-page-variable")
		assert.NotContains(t, result.Document, "variables-panel")
		assert.Contains(t, result.Document, "<style>")
		assert.Equal(t, "production", stub.snippets["variable:environment"].Content)
	})
	t.Run("rejects an unused variable with a field problem", func(t *testing.T) {
		t.Parallel()
		stub, _, logger := variableExportFixture(t)
		request := httptest.NewRequest(http.MethodPost, "/export/preview/guide", strings.NewReader(`{"variables":{"contact":"other"}}`))
		request.SetPathValue("slug", "guide")
		response := httptest.NewRecorder()
		PreviewPageExport(stub, stub, stub, stub, &exportMediaStub{}, emptyContractServices{}, md.New(), logger)(response, request)
		assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		assert.Contains(t, response.Body.String(), `"variables"`)
	})
	t.Run("rejects malformed input before querying dependencies", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"variables":[]}`))
		PreviewPageExport(nil, nil, nil, nil, nil, emptyContractServices{}, nil, nil)(response, request)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})
	t.Run("sanitizes markup in temporary values", func(t *testing.T) {
		t.Parallel()
		stub, _, logger := variableExportFixture(t)
		body, err := json.Marshal(exportVariablesRequest{Variables: map[string]string{"environment": `<script>alert(1)</script><img src="image.png" onerror="alert(1)">`}})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		request.SetPathValue("slug", "guide")
		response := httptest.NewRecorder()
		PreviewPageExport(stub, stub, stub, stub, &exportMediaStub{}, emptyContractServices{}, md.New(), logger)(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var result exportPreviewResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		assert.NotContains(t, result.Document, "<script")
		assert.NotContains(t, result.Document, "onerror")
	})
	t.Run("returns a missing page separately from render dependency failures", func(t *testing.T) {
		t.Parallel()
		stub, _, logger := variableExportFixture(t)
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		request.SetPathValue("slug", "missing")
		response := httptest.NewRecorder()
		PreviewPageExport(stub, stub, stub, stub, &exportMediaStub{}, emptyContractServices{}, md.New(), logger)(response, request)
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestPDFReceivesTemporaryVariables(t *testing.T) {
	t.Parallel()
	stub, views, logger := variableExportFixture(t)
	page := stub.pages["guide"]
	page.Markdown = "production {{var:environment}}\n\n{{subpages title=\"Related pages\"}}\n\n```text\n{{var:environment}}\n```"
	stub.pages["guide"] = page
	stub.navigationPages = []domain.Page{
		{Slug: "guide", Title: "Guide"},
		{Slug: "guide/install", Title: "Install"},
	}
	sent := make(chan string, 1)
	stub.pdfHeaders = []domain.PDFHeader{{Name: "Authorization", Value: "Bearer export-token"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer export-token", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		sent <- string(body)
		w.Header().Set("Content-Type", "application/pdf")
		_, err = io.WriteString(w, "%PDF-1.7\nfixture\n%%EOF\n")
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	stub.application.PDFURL = server.URL + "/render"
	request := httptest.NewRequest(http.MethodPost, "/export/pdf/guide", strings.NewReader(`{"variables":{"environment":"staging"}}`))
	request.SetPathValue("slug", "guide")
	response := httptest.NewRecorder()
	ExportPagePDF(stub, stub, stub, stub, &exportMediaStub{}, emptyContractServices{}, md.New(), views, logger)(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "application/pdf", response.Header().Get("Content-Type"))
	assert.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	sentHTML := <-sent
	assert.Contains(t, sentHTML, "production staging")
	assert.Contains(t, sentHTML, "Related pages")
	assert.Contains(t, sentHTML, `/pages/guide/install`)
	assert.Contains(t, sentHTML, "{{var:environment}}")
	assert.NotContains(t, sentHTML, "data-page-variable")
	assert.Equal(t, "production", stub.snippets["variable:environment"].Content)
}

func TestRenderedExportProblemClassification(t *testing.T) {
	t.Parallel()
	t.Run("missing knowledge is not reported as a missing page", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writeRenderedExportProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, domain.ErrNotFound)
		assert.Equal(t, http.StatusInternalServerError, response.Code)
	})
	t.Run("missing media retains its resource context", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writeRenderedExportProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, &exportMediaError{cause: domain.ErrNotFound})
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "image referenced")
	})
	t.Run("does not expose internal errors", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		writeRenderedExportProblem(slog.New(slog.NewTextHandler(io.Discard, nil)), response, errors.New("private-database-details"))
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.NotContains(t, response.Body.String(), "private-database-details")
	})
}

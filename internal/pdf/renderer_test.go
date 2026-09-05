package pdf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentEscapesThePageTitle(t *testing.T) {
	t.Parallel()

	result := document(`Runbooks <production>`, "de-CH", `<p>Content</p>`)
	assert.Contains(t, result, `<html lang="de-CH">`)
	assert.Contains(t, result, `<title>Runbooks &lt;production&gt;</title>`)
	assert.Contains(t, result, `<h1 class="document-title">Runbooks &lt;production&gt;</h1>`)
}

func TestDocumentExpandsInteractiveMarkdown(t *testing.T) {
	t.Parallel()

	result := document(
		"Runbook",
		"en",
		`<details class="markdown-details"><summary>Steps</summary><div class="markdown-tab-panel markdown-tab-panel-hidden">Deploy</div></details>`,
	)
	assert.NotContains(t, result, "<details")
	assert.NotContains(t, result, "<summary>")
	assert.NotContains(t, result, `class="markdown-tab-panel markdown-tab-panel-hidden"`)
	assert.Contains(t, result, `<div class="markdown-details-summary">Steps</div>`)
}

func TestRenderPostsToExactConfiguredEndpoint(t *testing.T) {
	t.Parallel()
	payload := "%PDF-1.7\nfixture\n%%EOF\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/custom/pdf/render?profile=wiki", r.URL.RequestURI())
		assert.Equal(t, "text/html; charset=utf-8", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/pdf", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `<html lang="de-CH">`)
		assert.Contains(t, string(body), `Title &lt;test&gt;`)
		assert.Contains(t, string(body), `data:image/png;base64,AAAA`)
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = io.WriteString(w, payload)
	}))
	defer server.Close()
	file, cleanup, err := Render(context.Background(), server.URL+"/custom/pdf/render?profile=wiki", "Title <test>", "de-CH", `<img src="data:image/png;base64,AAAA">`)
	require.NoError(t, err)
	defer cleanup()
	body, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, payload, string(body))
	name := file.Name()
	cleanup()
	_, err = os.Stat(name)
	assert.True(t, os.IsNotExist(err))
}

func TestRenderRejectsBadResponses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                      string
		status                    int
		contentType, body, length string
	}{
		{"upstream error", 500, "text/plain", "internal error", ""},
		{"redirect", 307, "application/pdf", "%PDF-1.7", ""},
		{"HTML response", 200, "text/html", "<html>error</html>", ""},
		{"invalid bytes", 200, "application/pdf", "not a pdf", ""},
		{"empty response", 200, "application/pdf", "", ""},
		{"oversized response", 200, "application/pdf", "%PDF-1.7", strconv.Itoa(maxPDFBytes + 1)},
		{"truncated response", 200, "application/pdf", "%PDF-1.7", "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/render" {
					t.Error("redirect must not be followed")
				}
				w.Header().Set("Location", "/unexpected")
				w.Header().Set("Content-Type", tc.contentType)
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			file, cleanup, err := Render(context.Background(), server.URL+"/render", "Title", "en", "<p>test</p>")
			cleanup()
			require.Error(t, err)
			assert.Nil(t, file)
		})
	}
}

func TestRenderCanceledAndUnconfigured(t *testing.T) {
	t.Parallel()
	_, cleanup, err := Render(context.Background(), "", "", "", "")
	cleanup()
	assert.ErrorIs(t, err, ErrNotConfigured)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, cleanup, err = Render(ctx, "http://127.0.0.1:1/render", "", "", "")
	cleanup()
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRenderBoundsUnknownLengthResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = io.WriteString(w, "%PDF-")
		w.(http.Flusher).Flush()
		chunk := strings.Repeat("x", 1<<20)
		for i := 0; i < 65; i++ {
			if _, err := io.WriteString(w, chunk); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	file, cleanup, err := Render(context.Background(), server.URL+"/render", "Title", "en", "<p>test</p>")
	cleanup()
	require.ErrorContains(t, err, "64 MiB")
	assert.Nil(t, file)
}

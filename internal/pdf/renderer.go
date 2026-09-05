// Package pdf sends standalone wiki documents to an external PDF renderer.
package pdf

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	maxHTMLBytes = 32 << 20
	maxPDFBytes  = 64 << 20
)

// ErrNotConfigured indicates PDF exports have no configured rendering endpoint.
var ErrNotConfigured = errors.New("PDF service is not configured")

var renderClient = &http.Client{
	Timeout: 60 * time.Second,
	// Never forward private wiki content to a redirect target or change POST to GET.
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// ValidateURL accepts an optional, complete HTTP endpoint including its path.
func ValidateURL(value string) error {
	if value == "" {
		return nil
	}
	endpoint, err := url.Parse(value)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Path == "" || endpoint.User != nil || endpoint.Fragment != "" {
		return errors.New("PDF URL must be an HTTP(S) endpoint including its path, without credentials or a fragment (for example http://pdf:8080/render)")
	}
	return nil
}

// Render POSTs HTML to endpoint exactly as configured and returns a temporary PDF.
// The caller must call cleanup after serving the file.
func Render(ctx context.Context, endpoint, title, language, rendered string) (*os.File, func(), error) {
	noop := func() {}
	if endpoint == "" {
		return nil, noop, ErrNotConfigured
	}
	if err := ValidateURL(endpoint); err != nil {
		return nil, noop, err
	}
	content := document(title, language, rendered)
	if len(content) > maxHTMLBytes {
		return nil, noop, errors.New("PDF document exceeds the 32 MiB request limit")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(content))
	if err != nil {
		return nil, noop, err
	}
	request.Header.Set("Content-Type", "text/html; charset=utf-8")
	request.Header.Set("Accept", "application/pdf")
	response, err := renderClient.Do(request)
	if err != nil {
		return nil, noop, fmt.Errorf("request PDF service: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, noop, fmt.Errorf("PDF service returned HTTP %d", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "application/pdf" {
		return nil, noop, errors.New("PDF service returned an unexpected content type")
	}
	if response.ContentLength > maxPDFBytes {
		return nil, noop, errors.New("PDF service response exceeds the 64 MiB limit")
	}
	prefix := make([]byte, 5)
	if _, err := io.ReadFull(response.Body, prefix); err != nil || string(prefix) != "%PDF-" {
		return nil, noop, errors.New("PDF service returned an invalid PDF")
	}
	file, err := os.CreateTemp("", "lore-pdf-*.pdf")
	if err != nil {
		return nil, noop, err
	}
	cleanup := func() { _ = file.Close(); _ = os.Remove(file.Name()) }
	size, err := io.Copy(file, io.LimitReader(io.MultiReader(strings.NewReader(string(prefix)), response.Body), maxPDFBytes+1))
	if err != nil {
		cleanup()
		return nil, noop, fmt.Errorf("read PDF service response: %w", err)
	}
	if size > maxPDFBytes {
		cleanup()
		return nil, noop, errors.New("PDF service response exceeds the 64 MiB limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, noop, err
	}
	return file, cleanup, nil
}

// document wraps rendered wiki HTML in a self-contained print-oriented document.
func document(title, language, rendered string) string {
	rendered = strings.ReplaceAll(rendered, " markdown-tab-panel-hidden", "")
	rendered = strings.ReplaceAll(
		rendered,
		`<details class="markdown-details"`,
		`<div class="markdown-details"`,
	)
	rendered = strings.ReplaceAll(rendered, "</details>", "</div>")
	rendered = strings.ReplaceAll(
		rendered,
		"<summary>",
		`<div class="markdown-details-summary">`,
	)
	rendered = strings.ReplaceAll(rendered, "</summary>", "</div>")

	title = html.EscapeString(title)
	language = html.EscapeString(language)

	return `<!doctype html>
<html lang="` + language + `">
<head>
	<meta charset="utf-8">
	<title>` + title + `</title>
	<style>` + styles + `</style>
</head>
<body>
	<main>
		<h1 class="document-title">` + title + `</h1>
		<article>` + rendered + `</article>
	</main>
</body>
</html>`
}

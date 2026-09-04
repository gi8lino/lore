// Package pdf renders standalone wiki documents with WeasyPrint.
package pdf

import (
	"context"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Render creates a temporary PDF whose returned cleanup function must be called after use.
func Render(ctx context.Context, title, language, rendered string) (*os.File, func(), error) {
	weasyprint, err := exec.LookPath("weasyprint")
	if err != nil {
		return nil, func() {}, fmt.Errorf("weasyprint executable not found: %w", err)
	}

	directory, err := os.MkdirTemp("", "lore-pdf-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }

	htmlPath := filepath.Join(directory, "page.html")
	pdfPath := filepath.Join(directory, "page.pdf")
	if err := os.WriteFile(htmlPath, []byte(document(title, language, rendered)), 0o600); err != nil {
		cleanup()
		return nil, func() {}, err
	}

	command := exec.CommandContext(ctx, weasyprint, htmlPath, pdfPath)
	if output, err := command.CombinedOutput(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("run weasyprint: %w: %s", err, strings.TrimSpace(string(output)))
	}

	file, err := os.Open(pdfPath)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return file, func() {
		_ = file.Close()
		cleanup()
	}, nil
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

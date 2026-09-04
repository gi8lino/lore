package pdf

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

package wasm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

// TestTaskListSyntaxRendersVisibleCheckboxes verifies the public task-list grammar owns its presentation renderer.
func TestTaskListSyntaxRendersVisibleCheckboxes(t *testing.T) {
	t.Parallel()

	markdown := goldmark.New(goldmark.WithExtensions(taskListSyntax{}))
	var output bytes.Buffer

	require.NoError(t, markdown.Convert([]byte("- [ ] pending\n- [x] completed\n"), &output))
	got := output.String()

	assert.Contains(t, got, `class="task-list-checkbox"`)
	assert.Contains(t, got, `aria-checked="false"`)
	assert.Contains(t, got, `class="task-list-checkbox checked"`)
	assert.Contains(t, got, `aria-checked="true"`)
	assert.NotContains(t, got, "<input")
}

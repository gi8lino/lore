package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskListsRenderVisibleCheckboxes(t *testing.T) {
	t.Parallel()

	renderer := New()
	got, err := renderer.Render("- [ ] pending\n- [x] completed\n")

	require.NoError(t, err)
	assert.Contains(t, got, `class="task-list-checkbox"`)
	assert.Contains(t, got, `aria-checked="false"`)
	assert.Contains(t, got, `class="task-list-checkbox checked"`)
	assert.Contains(t, got, `aria-checked="true"`)
	assert.NotContains(t, got, "<input")
}

func TestTaskListsCanBeDisabled(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.TaskLists = false
	got, err := renderer.RenderResolvedWithOptions("- [ ] pending\n", Slug, options)

	require.NoError(t, err)
	assert.NotContains(t, got, `class="task-list-checkbox`)
	assert.Contains(t, got, "[ ] pending")
}

func TestRawHTMLInputsRemainDisallowed(t *testing.T) {
	t.Parallel()

	renderer := New()
	got, err := renderer.Render(`<input type="checkbox" checked disabled>`)

	require.NoError(t, err)
	assert.NotContains(t, got, "<input")
}

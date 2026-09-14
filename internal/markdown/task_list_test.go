package markdown

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskListsRenderVisibleCheckboxes(t *testing.T) {
	t.Parallel()

	renderer := testRenderer(t)
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

	renderer := testRenderer(t)
	manager := renderer.PluginManager()
	require.NotNil(t, manager)
	require.NoError(t, manager.Disable(context.Background(), "io.lore.task-lists"))
	got, err := renderer.Render("- [ ] pending\n")

	require.NoError(t, err)
	assert.NotContains(t, got, `class="task-list-checkbox`)
	assert.Contains(t, got, "[ ] pending")
}

func TestRawHTMLInputsRemainDisallowed(t *testing.T) {
	t.Parallel()

	renderer := testRenderer(t)
	got, err := renderer.Render(`<input type="checkbox" checked disabled>`)

	require.NoError(t, err)
	assert.NotContains(t, got, "<input")
}

package themes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbeddedThemes(t *testing.T) {
	t.Parallel()

	available, err := Load("")
	require.NoError(t, err)

	t.Run("includes Light", func(t *testing.T) {
		t.Parallel()
		_, ok := Find(available, "Light")
		assert.True(t, ok)
	})
	t.Run("includes Dark", func(t *testing.T) {
		t.Parallel()
		_, ok := Find(available, "Dark")
		assert.True(t, ok)
	})
	t.Run("includes Catppuccin Mocha", func(t *testing.T) {
		t.Parallel()
		_, ok := Find(available, "Catppuccin Mocha")
		assert.True(t, ok)
	})
}

func TestFindIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	available, err := Load("")
	require.NoError(t, err)

	theme, ok := Find(available, "catppuccin mocha")
	require.True(t, ok)
	assert.Equal(t, "Catppuccin Mocha", theme.Title)
}

func TestLoadOverlaysThemeByFilename(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	custom := `color_scheme = "dark"

[colors]
background = "#000001"
surface = "#000002"
surface_elevated = "#000003"
surface_hover = "#000004"
text = "#ffffff"
text_secondary = "#eeeeee"
text_tertiary = "#dddddd"
muted = "#aaaaaa"
border = "#222222"
border_strong = "#333333"
border_subtle = "#111111"
accent = "#123456"
accent_secondary = "#234567"
accent_soft = "#345678"
success = "#456789"
warning = "#56789a"
error = "#6789ab"
danger = "#789abc"
selection_text = "#ffffff"
selection_background = "#123456"
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "Dark.toml"), []byte(custom), 0o600))

	available, err := Load(directory)
	require.NoError(t, err)
	theme, ok := Find(available, "dark")
	require.True(t, ok)
	assert.Equal(t, "#123456", theme.Colors.Accent)
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	custom := `color_scheme = "dark"
unknown = "value"
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "Broken.toml"), []byte(custom), 0o600))

	_, err := Load(directory)
	require.Error(t, err)
}

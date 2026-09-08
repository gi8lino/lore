package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRejectsOverlappingSourceAndOutput(t *testing.T) {
	t.Parallel()

	t.Run("output inside source", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.SourceDir = filepath.Join(root, "docs")
		config.OutputDir = filepath.Join(root, "docs", "site")
		assert.Error(t, config.validate())
	})

	t.Run("source inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.SourceDir = filepath.Join(root, "site", "docs")
		config.OutputDir = filepath.Join(root, "site")
		assert.Error(t, config.validate())
	})
}

func TestBrandingPathsRelativeToConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets"), 0o755))
	for _, name := range []string{"logo.svg", "favicon.svg", "favicon.ico"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "assets", name), []byte("image"), 0o644))
	}
	filename := filepath.Join(root, "lore-site.toml")
	require.NoError(t, os.WriteFile(filename, []byte(`logo = "assets/logo.svg"
favicon = "assets/favicon.svg"
favicon_ico = "assets/favicon.ico"
assets_dir = "assets"
`), 0o644))
	config, err := LoadConfig(filename, true)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "assets", "logo.svg"), config.Logo)
	require.Equal(t, filepath.Join(root, "assets", "favicon.svg"), config.Favicon)
	require.Equal(t, filepath.Join(root, "assets", "favicon.ico"), config.FaviconICO)
	require.Equal(t, filepath.Join(root, "assets"), config.AssetsDir)
	require.Equal(t, "docs", config.SourceDir)
}

func TestBrandingValidation(t *testing.T) {
	t.Parallel()

	t.Run("logo", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Logo = filepath.Join(root, "missing")
		assert.ErrorContains(t, config.validate(), "logo")

		config.Logo = filepath.Join(config.OutputDir, "image.svg")
		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Favicon = filepath.Join(root, "missing")
		assert.ErrorContains(t, config.validate(), "favicon")

		config.Favicon = filepath.Join(config.OutputDir, "image.svg")
		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon_ico", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.FaviconICO = filepath.Join(root, "missing")
		assert.ErrorContains(t, config.validate(), "favicon_ico")

		config.FaviconICO = filepath.Join(config.OutputDir, "image.svg")
		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("assets_dir", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.AssetsDir = filepath.Join(root, "missing")
		assert.ErrorContains(t, config.validate(), "assets_dir")

		config.AssetsDir = filepath.Join(config.OutputDir, "image.svg")
		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})
}

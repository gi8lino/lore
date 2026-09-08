package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containeroo/tinyflags"
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
	assetsDir := filepath.Join(root, "assets")
	contentDir := filepath.Join(root, "content")
	require.NoError(t, os.MkdirAll(assetsDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(contentDir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "never.svg"), []byte("logo"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(contentDir, "assets", "favicon.svg"), []byte("favicon"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(contentDir, "favicon.ico"), []byte("favicon ico"), 0o644))

	filename := filepath.Join(root, "lore-site.toml")
	require.NoError(t, os.WriteFile(filename, []byte(`logo = "assets/never.svg"
favicon = "content/assets/favicon.svg"
favicon_ico = "content/favicon.ico"
assets_dir = "assets"
`), 0o644))

	config, err := LoadConfig(filename, true)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "assets", "never.svg"), config.Logo)
	assert.Equal(t, filepath.Join(root, "content", "assets", "favicon.svg"), config.Favicon)
	assert.Equal(t, filepath.Join(root, "content", "favicon.ico"), config.FaviconICO)
	assert.Equal(t, filepath.Join(root, "assets"), config.AssetsDir)
}

func TestBrandingValidation(t *testing.T) {
	t.Parallel()

	t.Run("logo missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Logo = filepath.Join(root, "missing.svg")

		assert.ErrorContains(t, config.validate(), "logo")
	})

	t.Run("logo inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Logo = filepath.Join(config.OutputDir, "image.svg")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Favicon = filepath.Join(root, "missing.svg")

		assert.ErrorContains(t, config.validate(), "favicon")
	})

	t.Run("favicon inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Favicon = filepath.Join(config.OutputDir, "image.svg")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon ico missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.FaviconICO = filepath.Join(root, "missing.ico")

		assert.ErrorContains(t, config.validate(), "favicon_ico")
	})

	t.Run("favicon ico inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.FaviconICO = filepath.Join(config.OutputDir, "favicon.ico")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("assets directory missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.AssetsDir = filepath.Join(root, "missing")

		assert.ErrorContains(t, config.validate(), "assets_dir")
	})

	t.Run("assets directory contains output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := DefaultConfig()
		config.AssetsDir = filepath.Join(root, "assets")
		config.OutputDir = filepath.Join(config.AssetsDir, "site")
		require.NoError(t, os.MkdirAll(config.AssetsDir, 0o755))

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})
}

func TestBuildFlagsOverrideConfigurationFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "site.toml")

	require.NoError(t, os.WriteFile(configPath, []byte(`
site_name = "From file"
site_url = "https://example.com/docs/"
source_dir = "docs"
output_dir = "site"
theme = "Light"
language = "en"
mermaid = true
`), 0o600))

	flags := tinyflags.NewFlagSet("lore build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--config", configPath,
		"--site-name", "From CLI",
		"--mermaid=false",
	}))

	cfg, err := resolve()

	require.NoError(t, err)
	assert.Equal(t, "From CLI", cfg.SiteName)
	assert.Equal(t, "https://example.com/docs/", cfg.SiteURL)
	assert.False(t, cfg.Mermaid)
}

func TestBuildFlagsRejectInvalidDirectoryOverride(t *testing.T) {
	t.Parallel()

	flags := tinyflags.NewFlagSet("lore build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--source", "docs",
		"--output", "docs",
	}))

	_, err := resolve()

	assert.ErrorContains(t, err, "source_dir and output_dir must be separate directories")
}

package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigRejectsOverlappingSourceAndOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	config := DefaultConfig()
	config.SourceDir = filepath.Join(root, "docs")
	config.OutputDir = filepath.Join(root, "docs", "site")

	require.Error(t, config.validate())

	config.SourceDir = filepath.Join(root, "site", "docs")
	config.OutputDir = filepath.Join(root, "site")

	require.Error(t, config.validate())
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
	root := t.TempDir()
	for _, field := range []string{"logo", "favicon", "favicon_ico", "assets_dir"} {
		t.Run(field, func(t *testing.T) {
			config := DefaultConfig()
			config.OutputDir = filepath.Join(root, "output")
			fields := map[string]*string{"logo": &config.Logo, "favicon": &config.Favicon, "favicon_ico": &config.FaviconICO, "assets_dir": &config.AssetsDir}
			*fields[field] = filepath.Join(root, "missing")
			require.ErrorContains(t, config.validate(), field)
			*fields[field] = filepath.Join(config.OutputDir, "image.svg")
			require.ErrorContains(t, config.validate(), "separate from output_dir")
		})
	}
}

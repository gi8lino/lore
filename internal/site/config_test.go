package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRejectsOverlappingSourceAndOutput(t *testing.T) {
	t.Parallel()

	t.Run("output inside source", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.SourceDir = filepath.Join(root, "docs")
		config.OutputDir = filepath.Join(root, "docs", "site")

		assert.Error(t, config.validate())
	})

	t.Run("source inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
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

	config, err := loadConfig(filename, true)

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
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Logo = filepath.Join(root, "missing.svg")

		assert.ErrorContains(t, config.validate(), "logo")
	})

	t.Run("logo inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Logo = filepath.Join(config.OutputDir, "image.svg")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Favicon = filepath.Join(root, "missing.svg")

		assert.ErrorContains(t, config.validate(), "favicon")
	})

	t.Run("favicon inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.Favicon = filepath.Join(config.OutputDir, "image.svg")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("favicon ico missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.FaviconICO = filepath.Join(root, "missing.ico")

		assert.ErrorContains(t, config.validate(), "favicon_ico")
	})

	t.Run("favicon ico inside output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.FaviconICO = filepath.Join(config.OutputDir, "favicon.ico")

		assert.ErrorContains(t, config.validate(), "separate from output_dir")
	})

	t.Run("assets directory missing", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
		config.OutputDir = filepath.Join(root, "output")
		config.AssetsDir = filepath.Join(root, "missing")

		assert.ErrorContains(t, config.validate(), "assets_dir")
	})

	t.Run("assets directory contains output", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		config := defaultConfig()
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
navigation_style = "tree"
navigation_density = "compact"
sidebar_width = 360
mermaid = true
`), 0o600))

	flags := tinyflags.NewFlagSet("lore build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--config", configPath,
		"--site-name", "From CLI",
		"--navigation-style", "topbar",
		"--navigation-density", "comfortable",
		"--sidebar-width", "320",
		"--mermaid=false",
		"--robots=disallow",
	}))

	cfg, err := resolve()

	require.NoError(t, err)
	assert.Equal(t, "From CLI", cfg.SiteName)
	assert.Equal(t, "https://example.com/docs/", cfg.SiteURL)
	assert.Equal(t, domain.NavigationStyleTopbar, cfg.NavigationStyle)
	assert.Equal(t, domain.NavigationDensityComfortable, cfg.NavigationDensity)
	assert.Equal(t, 320, cfg.SidebarWidth)
	assert.False(t, cfg.Mermaid)
	assert.Equal(t, domain.RobotsPolicyDisallow, cfg.RobotsPolicy)
}

func TestDefaultConfigUsesGenericBranding(t *testing.T) {
	t.Parallel()

	config := defaultConfig()

	assert.Equal(t, "Documentation", config.SiteName)
	assert.Empty(t, config.Logo)
	assert.Empty(t, config.Favicon)
	assert.Empty(t, config.FaviconICO)
	assert.Equal(t, domain.NavigationStyleSidebar, config.NavigationStyle)
	assert.Equal(t, domain.NavigationDensityComfortable, config.NavigationDensity)
	assert.Equal(t, domain.DefaultSidebarWidth, config.SidebarWidth)
	assert.Equal(t, domain.RobotsPolicyAllow, config.RobotsPolicy)
}

func TestNavigationPresentationConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("loads presentation settings", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		filename := filepath.Join(root, "site.toml")
		require.NoError(t, os.WriteFile(filename, []byte(`
navigation_style = "tree"
navigation_density = "compact"
sidebar_width = 360
`), 0o600))

		config, err := loadConfig(filename, true)

		require.NoError(t, err)
		assert.Equal(t, domain.NavigationStyleTree, config.NavigationStyle)
		assert.Equal(t, domain.NavigationDensityCompact, config.NavigationDensity)
		assert.Equal(t, 360, config.SidebarWidth)
	})

	t.Run("rejects unknown navigation style", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.NavigationStyle = "columns"

		assert.ErrorContains(t, config.validate(), "navigation_style must be sidebar, topbar, or tree")
	})

	t.Run("rejects unknown navigation density", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.NavigationDensity = "dense"

		assert.ErrorContains(t, config.validate(), "navigation_density must be comfortable or compact")
	})

	t.Run("rejects sidebar width below range", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.SidebarWidth = domain.MinSidebarWidth - 1

		assert.ErrorContains(t, config.validate(), "sidebar_width must be between 220 and 420 pixels")
	})

	t.Run("rejects sidebar width above range", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.SidebarWidth = domain.MaxSidebarWidth + 1

		assert.ErrorContains(t, config.validate(), "sidebar_width must be between 220 and 420 pixels")
	})
}

func TestRobotsConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("loads policy", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		filename := filepath.Join(root, "site.toml")
		require.NoError(t, os.WriteFile(filename, []byte(`robots = "none"`), 0o600))

		config, err := loadConfig(filename, true)

		require.NoError(t, err)
		assert.Equal(t, domain.RobotsPolicyNone, config.RobotsPolicy)
	})

	t.Run("rejects unknown policy", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.RobotsPolicy = "sometimes"

		assert.ErrorContains(t, config.validate(), "robots must be allow, disallow, or none")
	})
}

func TestBuildFlagsValidateOverrides(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	flags := tinyflags.NewFlagSet("lore build", tinyflags.ContinueOnError)
	resolve := BindFlags(flags)

	require.NoError(t, flags.Parse([]string{
		"--source", filepath.Join(root, "docs"),
		"--output", filepath.Join(root, "docs", "site"),
	}))

	_, err := resolve()

	assert.ErrorContains(t, err, "source_dir and output_dir must be separate directories")
}
func TestExternalLinkConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("loads and normalizes links", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		filename := filepath.Join(root, "site.toml")
		require.NoError(t, os.WriteFile(filename, []byte(`
[[external_links]]
label = " Repository "
url = " https://github.com/gi8lino/lore "
icon = " github-simple "
description = " v2.4.1 "
hover_effect = " lift "
hover_text = " {{label }} | {{description}} "
`), 0o600))

		config, err := loadConfig(filename, true)

		require.NoError(t, err)
		require.Len(t, config.ExternalLinks, 1)
		assert.Equal(t, "Repository", config.ExternalLinks[0].Label)
		assert.Equal(t, "https://github.com/gi8lino/lore", config.ExternalLinks[0].URL)
		assert.Equal(t, "github-simple", config.ExternalLinks[0].Icon)
		assert.Equal(t, "v2.4.1", config.ExternalLinks[0].Description)
		assert.Equal(t, domain.ExternalLinkHoverLift, config.ExternalLinks[0].HoverEffect)
		assert.Equal(t, "{{label }} | {{description}}", config.ExternalLinks[0].HoverText)
	})

	t.Run("rejects unsafe URL", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.ExternalLinks = []domain.ExternalLink{{Label: "Repository", URL: "javascript:alert(1)"}}

		assert.ErrorContains(t, config.validate(), "HTTP or HTTPS")
	})

	t.Run("rejects unknown icon", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.ExternalLinks = []domain.ExternalLink{{Label: "Repository", URL: "https://example.test", Icon: "not-an-icon"}}

		assert.ErrorContains(t, config.validate(), "available icon")
	})

	t.Run("rejects unknown hover effect", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.ExternalLinks = []domain.ExternalLink{{Label: "Repository", URL: "https://example.test", HoverEffect: "bounce"}}

		assert.ErrorContains(t, config.validate(), "hover_effect")
	})
}

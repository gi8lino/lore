package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowsRootHelpWhenCommandIsMissing(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), nil, nil, "test", "deadbeef", &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "serve")
	assert.Contains(t, stdout.String(), "site")
}

func TestRunShowsServeHelpWithoutRuntimeConfiguration(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"serve", "--help"}, nil, "test", "deadbeef", &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "--database-url")
	assert.Contains(t, stdout.String(), "--listen-address")
}

func TestRunShowsNestedSiteHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"site"}, nil, "test", "deadbeef", &stdout)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "build")
}

func TestSiteBuildFlagsOverrideConfigurationFile(t *testing.T) {
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

	flags := tinyflags.NewFlagSet("lore site build", tinyflags.ContinueOnError)
	resolve := bindSiteBuildFlags(flags)
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

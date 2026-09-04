package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/containeroo/tinyflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowsRootHelpWhenCommandIsMissing(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := Run(context.Background(), nil, nil, "test", "deadbeef", io.Discard, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "serve")
	assert.Contains(t, stderr.String(), "build")
}

func TestRunShowsServeHelpWithoutRuntimeConfiguration(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"serve", "--help"}, nil, "test", "deadbeef", &stdout, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "--database-url")
	assert.Contains(t, stdout.String(), "--listen-address")
}

func TestRunShowsBuildHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"build", "--help"}, nil, "test", "deadbeef", &stdout, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "--config")
	assert.Contains(t, stdout.String(), "--site-name")
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
	resolve := bindBuildFlags(flags)
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

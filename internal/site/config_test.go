package site

import (
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

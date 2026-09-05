package handler

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImportFormatRequiresExplicitFormat(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "auto", "json"} {
		_, err := parseImportFormat(value)
		require.Error(t, err)
	}

	format, err := parseImportFormat("wikijs")

	require.NoError(t, err)
	assert.Equal(t, wikiJSImport, format)
}

func TestImportWikiJSONRequiresExplicitPageFields(t *testing.T) {
	t.Parallel()

	pages, err := importWikiJSON([]byte(`[
  {"path":"guides/setup","title":"Setup","content":"# Setup\n"}
]`))

	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "guides/setup", pages[0].Slug)
	assert.Equal(t, "Setup", pages[0].Title)

	tests := []string{
		`{"pages":[{"path":"guide","title":"Guide","content":"# Guide"}]}`,
		`[{"title":"Guide","content":"# Guide"}]`,
		`[{"path":"guide","content":"# Guide"}]`,
		`[{"path":"guide","title":"Guide"}]`,
	}

	for _, input := range tests {
		_, err := importWikiJSON([]byte(input))
		require.Error(t, err)
	}
}

func TestMarkdownTitleRequiresExplicitHeading(t *testing.T) {
	t.Parallel()

	title, err := markdownTitle("intro\n# Explicit title\n")

	require.NoError(t, err)
	assert.Equal(t, "Explicit title", title)

	_, err = markdownTitle("No title here")

	require.Error(t, err)
}

func TestReadImportArchiveEntryEnforcesRemainingBudget(t *testing.T) {
	t.Parallel()

	_, err := readImportArchiveEntry(io.NopCloser(strings.NewReader("abc")), 2)

	require.EqualError(t, err, "archive contents exceed 100 MiB")
}

func TestImportZIPBudgetSharedAcrossFiles(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("page.md")
	require.NoError(t, err)
	_, err = entry.Write([]byte("# Title"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	remaining := int64(10)
	items, err := importZIP(archive.Bytes(), markdownImport, &remaining)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3), remaining)
	_, err = importZIP(archive.Bytes(), markdownImport, &remaining)
	require.EqualError(t, err, "archive contents exceed 100 MiB")
}

package pluginpackage

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testManifest = `api_version: 1
id: io.example.test
name: Test
version: 1.0.0
modules:
  - type: renderer-extension
    id: test
    stage: preprocess
permissions: []
`

func testArchive(t *testing.T, manifest string, extras ...archiveEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := append([]archiveEntry{{"plugin.yaml", []byte(manifest), 0644}, {"plugin.wasm", []byte{0, 'a', 's', 'm', 1, 0, 0, 0}, 0644}}, extras...)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		writer, err := archive.CreateHeader(header)
		require.NoError(t, err)
		_, err = writer.Write(entry.data)
		require.NoError(t, err)
	}
	require.NoError(t, archive.Close())
	return buffer.Bytes()
}

type archiveEntry struct {
	name string
	data []byte
	mode fs.FileMode
}

func TestPackageReadAndDefensiveCopies(t *testing.T) {
	data := testArchive(t, testManifest, archiveEntry{"assets/plugin.css", []byte("body{}"), 0644})
	pkg, err := Read(data)
	require.NoError(t, err)
	assert.Equal(t, "io.example.test", pkg.Manifest().ID)
	assert.Equal(t, []string{"plugin.css"}, pkg.AssetNames())
	content, err := pkg.Asset("plugin.css")
	require.NoError(t, err)
	content[0] = '!'
	original, err := pkg.Asset("plugin.css")
	require.NoError(t, err)
	assert.Equal(t, "body{}", string(original))
	manifest := pkg.Manifest()
	manifest.Modules[0].ID = "changed"
	assert.Equal(t, "test", pkg.Manifest().Modules[0].ID)
	wasm := pkg.WASM()
	wasm[0] = 1
	assert.Equal(t, byte(0), pkg.WASM()[0])
	_, err = pkg.Asset("../plugin.wasm")
	assert.ErrorIs(t, err, fs.ErrInvalid)
	_, err = pkg.Asset("missing")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestPackageRejectsUnsafeEntries(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "assets/../escape", "assets\\escape", "assets/a/../../escape", "C:/escape", "assets/./file", "other.txt", "plugin.yaml"} {
		t.Run(name, func(t *testing.T) {
			_, err := Read(testArchive(t, testManifest, archiveEntry{name, []byte("bad"), 0644}))
			require.Error(t, err)
		})
	}
	for _, mode := range []fs.FileMode{fs.ModeSymlink | 0644, fs.ModeNamedPipe | 0644} {
		_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/link", []byte("/etc/passwd"), mode}))
		require.Error(t, err, "mode %v", mode)
	}
	_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/link/", nil, fs.ModeSymlink | 0644}))
	require.Error(t, err)
	_, err = Read(testArchive(t, testManifest, archiveEntry{"assets/parent", nil, 0644}, archiveEntry{"assets/parent/child", nil, 0644}))
	require.Error(t, err)
}

func TestPackageRejectsUnsupportedManifest(t *testing.T) {
	for _, manifest := range []string{
		strings.Replace(testManifest, "api_version: 1", "api_version: 999", 1),
		strings.Replace(testManifest, "permissions: []", "permissions: [network]", 1),
		strings.Replace(testManifest, "stage: preprocess", "stage: native", 1),
		strings.Replace(testManifest, "renderer-extension", "database", 1),
		strings.Replace(testManifest, "1.0.0", "latest", 1),
		testManifest + "unknown: value\n",
		testManifest + "api_version: 1\n",
		testManifest + "---\nname: extra\n",
		testManifest + "requires: [io.example.test]\n",
	} {
		_, err := Read(testArchive(t, manifest))
		require.Error(t, err, manifest)
	}
}

func TestPackageRejectsExpansionAndEntryCountLimits(t *testing.T) {
	_, err := Read(testArchive(t, testManifest, archiveEntry{"assets/large.css", bytes.Repeat([]byte{'x'}, MaxAssetBytes+1), 0644}))
	require.ErrorContains(t, err, "size limit")
	entries := make([]archiveEntry, MaxFiles)
	for i := range entries {
		entries[i] = archiveEntry{"assets/" + strings.Repeat("x", i+1), nil, 0644}
	}
	_, err = Read(testArchive(t, testManifest, entries...))
	require.ErrorContains(t, err, "too many")
	_, err = Read(make([]byte, MaxArchiveBytes+1))
	require.Error(t, err)
}

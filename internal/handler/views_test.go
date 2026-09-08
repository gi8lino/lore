package handler

import (
	"testing"
	"testing/fstest"

	"github.com/gi8lino/lore/internal/navigation"
	"github.com/stretchr/testify/assert"
)

func TestActiveNavigationSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "applications/identity/keycloak", activeNavigationSlug("/pages/applications/identity/keycloak"))
	assert.Equal(t, "platforms/kubernetes", activeNavigationSlug("/edit/platforms/kubernetes"))
	assert.Empty(t, activeNavigationSlug("/pages/new"))
	assert.Empty(t, activeNavigationSlug("/settings"))
}

func TestPagePathOptions(t *testing.T) {
	t.Parallel()

	tree := []navigation.Node{
		{
			Slug: "platforms", Title: "Platforms", Children: []navigation.Node{
				{Slug: "platforms/containers", Title: "Containers"},
				{Slug: "platforms/kubernetes", Title: "Kubernetes"},
			},
		},
	}

	assert.Equal(t, []pagePathOption{
		{Slug: "platforms", Label: "Platforms"},
		{Slug: "platforms/containers", Label: "Platforms / Containers"},
		{Slug: "platforms/kubernetes", Label: "Platforms / Kubernetes"},
	}, pagePathOptions(tree, ""))
	assert.Equal(t, []pagePathOption{
		{Slug: "platforms", Label: "Platforms"},
		{Slug: "platforms/kubernetes", Label: "Platforms / Kubernetes"},
	}, pagePathOptions(tree, "platforms/containers"))
}

func TestSplitPagePath(t *testing.T) {
	t.Parallel()

	parent, segment := splitPagePath(" platforms/containers/docker ")
	assert.Equal(t, "platforms/containers", parent)
	assert.Equal(t, "docker", segment)

	parent, segment = splitPagePath("reference")
	assert.Empty(t, parent)
	assert.Equal(t, "reference", segment)
}

// TestFingerprintAssets verifies embedded asset fingerprints are stable and content-sensitive.
func TestFingerprintAssets(t *testing.T) {
	t.Parallel()

	first := fstest.MapFS{
		"css/app.css": &fstest.MapFile{Data: []byte("body{}")},
		"js/main.js":  &fstest.MapFile{Data: []byte("console.log('one')")},
	}
	second := fstest.MapFS{
		"css/app.css": &fstest.MapFile{Data: []byte("body{}")},
		"js/main.js":  &fstest.MapFile{Data: []byte("console.log('two')")},
	}

	firstVersion, err := fingerprintAssets(first)

	assert.NoError(t, err)

	repeatedVersion, err := fingerprintAssets(first)

	assert.NoError(t, err)

	secondVersion, err := fingerprintAssets(second)

	assert.NoError(t, err)

	assert.Len(t, firstVersion, 16)
	assert.Equal(t, firstVersion, repeatedVersion)
	assert.NotEqual(t, firstVersion, secondVersion)
}

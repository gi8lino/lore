package handler

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestActiveNavigationSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "applications/identity/keycloak", activeNavigationSlug("/pages/applications/identity/keycloak"))
	assert.Equal(t, "platforms/kubernetes", activeNavigationSlug("/edit/platforms/kubernetes"))
	assert.Empty(t, activeNavigationSlug("/settings"))
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

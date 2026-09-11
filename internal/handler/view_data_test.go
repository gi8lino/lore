package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActiveNavigationSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "applications/identity/keycloak", activeNavigationSlug("/pages/applications/identity/keycloak"))
	assert.Equal(t, "platforms/kubernetes", activeNavigationSlug("/edit/platforms/kubernetes"))
	assert.Empty(t, activeNavigationSlug("/pages/new"))
	assert.Empty(t, activeNavigationSlug("/settings"))
}

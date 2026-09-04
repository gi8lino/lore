package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstHeader(t *testing.T) {
	t.Parallel()

	t.Run("uses the first populated configured header", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest("GET", "/", nil)
		request.Header.Set("X-Secondary-User", "daniel")
		request.Header.Set("X-Tertiary-User", "ignored")

		assert.Equal(t, "daniel", firstHeader(request, []string{
			"X-Primary-User",
			"X-Secondary-User",
			"X-Tertiary-User",
		}))
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest("GET", "/", nil)
		request.Header.Set("X-User", "  daniel  ")

		assert.Equal(t, "daniel", firstHeader(request, []string{"X-User"}))
	})

	t.Run("returns empty when no configured header is populated", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest("GET", "/", nil)
		assert.Empty(t, firstHeader(request, []string{"X-User"}))
	})
}

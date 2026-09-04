package handler

import (
	"testing"

	"github.com/gi8lino/lore/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeImageFilename(t *testing.T) {
	t.Parallel()

	t.Run("keeps safe png", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "architecture.png", service.SanitizeImageFilename("architecture.png", "image/png"))
	})
	t.Run("normalizes spaces", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "my-diagram.png", service.SanitizeImageFilename("my diagram.png", "image/png"))
	})
	t.Run("corrects extension", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "photo.jpg", service.SanitizeImageFilename("photo.png", "image/jpeg"))
	})
	t.Run("allows jpeg extension", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "photo.jpeg", service.SanitizeImageFilename("photo.jpeg", "image/jpeg"))
	})
	t.Run("adds extension", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "diagram.webp", service.SanitizeImageFilename("diagram", "image/webp"))
	})
}

func TestSupportedImageType(t *testing.T) {
	t.Parallel()

	t.Run("supports JPEG", func(t *testing.T) {
		t.Parallel()
		assert.True(t, service.SupportedImageType("image/jpeg"))
	})
	t.Run("supports PNG", func(t *testing.T) {
		t.Parallel()
		assert.True(t, service.SupportedImageType("image/png"))
	})
	t.Run("supports GIF", func(t *testing.T) {
		t.Parallel()
		assert.True(t, service.SupportedImageType("image/gif"))
	})
	t.Run("supports WebP", func(t *testing.T) {
		t.Parallel()
		assert.True(t, service.SupportedImageType("image/webp"))
	})
	t.Run("rejects SVG", func(t *testing.T) {
		t.Parallel()
		assert.False(t, service.SupportedImageType("image/svg+xml"))
	})
}

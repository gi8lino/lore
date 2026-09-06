package ascii

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAlphanumeric(t *testing.T) {
	t.Parallel()

	for _, character := range []rune{'a', 'z', 'A', 'Z', '0', '9'} {
		assert.True(t, IsAlphanumeric(character), "expected %q to be alphanumeric", character)
	}

	for _, character := range []rune{'-', '_', '.', '/', ' ', '\u00e9'} {
		assert.False(t, IsAlphanumeric(character), "expected %q not to be alphanumeric", character)
	}
}

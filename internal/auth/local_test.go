package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidLocalPassword(t *testing.T) {
	t.Parallel()

	assert.False(t, ValidLocalPassword("short"))
	assert.True(t, ValidLocalPassword("twelve-chars!"))
}

func TestLocalPasswordHash(t *testing.T) {
	t.Parallel()

	hash, err := localPasswordHash("correct-horse-battery-staple")

	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct-horse-battery-staple")))
	assert.Error(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong-password")))
}

func TestLocalSessionHash(t *testing.T) {
	t.Parallel()

	assert.Equal(t, localSessionHash("token"), localSessionHash("token"))
	assert.NotEqual(t, localSessionHash("token"), localSessionHash("other"))
}

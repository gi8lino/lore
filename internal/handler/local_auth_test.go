package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeAuthNext(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/admin/configuration", safeAuthNext(" /admin/configuration "))
	assert.Empty(t, safeAuthNext("https://example.com"))
	assert.Empty(t, safeAuthNext("//example.com"))
}

func TestSetupValidationError(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"username":         {"admin"},
		"password":         {"correct-horse-battery-staple"},
		"password_confirm": {"correct-horse-battery-staple"},
	}
	request := httptest.NewRequest("POST", "/setup", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, request.ParseForm())
	assert.Empty(t, setupValidationError(request))

	request.Form.Set("password_confirm", "different-password")
	assert.Equal(t, "Passwords do not match.", setupValidationError(request))
}

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type setupBrowserRepository struct {
	browserRepository
	settings               domain.ApplicationSettings
	setupRequired          bool
	localAdminCredential   bool
	localCredentialChecked bool
}

func (r *setupBrowserRepository) ApplicationSettings(context.Context) (domain.ApplicationSettings, error) {
	return r.settings, nil
}

func (r *setupBrowserRepository) SetupRequired(context.Context) (bool, error) {
	return r.setupRequired, nil
}

func (r *setupBrowserRepository) HasLocalAdministratorCredential(context.Context) (bool, error) {
	r.localCredentialChecked = true
	return r.localAdminCredential, nil
}

func TestConfigureBrowserAuthAllowsSetupWithStaleLocalMode(t *testing.T) {
	t.Parallel()

	repository := &setupBrowserRepository{
		settings: domain.ApplicationSettings{
			Authentication: domain.AuthenticationSettings{Mode: string(AuthModeLocal)},
		},
		setupRequired: true,
	}

	configured, err := ConfigureBrowserAuth(context.Background(), BrowserConfig{}, repository)

	require.NoError(t, err)
	assert.NotNil(t, configured.Authenticator)
	assert.False(t, repository.localCredentialChecked)
}

func TestBrowserLoginRedirectsSetupWithStaleLocalMode(t *testing.T) {
	t.Parallel()

	repository := &setupBrowserRepository{
		settings: domain.ApplicationSettings{
			Authentication: domain.AuthenticationSettings{Mode: string(AuthModeLocal)},
		},
		setupRequired: true,
	}
	browser := &browserAuthenticator{repository: repository}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/login", nil)

	browser.login(response, request)

	assert.Equal(t, http.StatusFound, response.Code)
	assert.Equal(t, "/setup", response.Header().Get("Location"))
}

func TestBrowserValidationStillRequiresLocalAdministratorAfterSetup(t *testing.T) {
	t.Parallel()

	repository := &setupBrowserRepository{setupRequired: false}
	browser := &browserAuthenticator{repository: repository}

	err := browser.validate(context.Background(), domain.AuthenticationSettings{Mode: string(AuthModeLocal)})

	validation, ok := errors.AsType[*domain.ValidationError](err)
	require.True(t, ok)
	assert.Equal(t, "auth_mode", validation.Fields[0].Field)
	assert.Equal(t, "Local authentication requires an administrator with a local password.", validation.UserMessage())
	assert.True(t, repository.localCredentialChecked)
}

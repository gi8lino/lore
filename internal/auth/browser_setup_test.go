package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type setupBrowserRepository struct {
	browserRepository
	settings               model.ApplicationSettings
	setupRequired          bool
	localAdminCredential   bool
	localCredentialChecked bool
}

func (r *setupBrowserRepository) ApplicationSettings(context.Context) (model.ApplicationSettings, error) {
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
		settings: model.ApplicationSettings{
			Authentication: model.AuthenticationSettings{Mode: string(AuthModeLocal)},
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
		settings: model.ApplicationSettings{
			Authentication: model.AuthenticationSettings{Mode: string(AuthModeLocal)},
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

	err := browser.validate(context.Background(), model.AuthenticationSettings{Mode: string(AuthModeLocal)})

	assert.EqualError(t, err, "local authentication requires an administrator with a local password")
	assert.True(t, repository.localCredentialChecked)
}

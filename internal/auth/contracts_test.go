package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gi8lino/lore/internal/domain"
)

// Keep every implementation of the shared authentication boundary checked.
var (
	_ Authenticator = (*Bearer)(nil)
	_ Authenticator = (*Local)(nil)
	_ Authenticator = (*OIDC)(nil)
	_ Authenticator = (*None)(nil)
	_ Authenticator = (*TrustedProxy)(nil)
	_ Authenticator = (*browserAuthenticator)(nil)
)

type authenticationContractRepository struct {
	localRepository
	oidcRepository
	user domain.User
}

func (s authenticationContractRepository) UserByToken(context.Context, string) (domain.User, error) {
	return s.user, nil
}
func (s authenticationContractRepository) LocalUserBySession(context.Context, string) (domain.User, error) {
	return s.user, nil
}
func (s authenticationContractRepository) LocalCredential(context.Context, string) (domain.User, string, error) {
	return s.user, "", nil
}
func (s authenticationContractRepository) OIDCUser(context.Context, string, string) (domain.User, error) {
	return s.user, nil
}

func TestAuthenticationEnabledAccountContract(t *testing.T) {
	t.Parallel()
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repository := authenticationContractRepository{user: domain.User{ID: 7, Enabled: enabled, SessionVersion: 1}}
			local := NewLocal(repository, "https://example.test")
			oidc := &OIDC{repository: repository, issuer: "https://identity.example.test", secret: []byte("0123456789abcdef0123456789abcdef")}
			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set("Authorization", "Bearer token")
			request.AddCookie(&http.Cookie{Name: localSessionCookie, Value: "token"})
			recorder := httptest.NewRecorder()
			oidc.setCookie(recorder, "lore_session", session{Issuer: oidc.issuer, Subject: "subject", Version: 1, Expires: time.Now().Add(time.Hour).Unix()}, 3600)
			request.AddCookie(recorder.Result().Cookies()[0])
			for _, tt := range []struct {
				name          string
				authenticator Authenticator
				denied        error
			}{
				{"bearer", NewBearer(repository), ErrInvalidCredentials},
				{"local", local, ErrUnauthenticated},
				{"oidc", oidc, ErrUnauthenticated},
			} {
				t.Run(tt.name, func(t *testing.T) {
					user, err := tt.authenticator.Authenticate(request)
					if enabled {
						if err != nil || user.ID != 7 {
							t.Fatalf("enabled user = %#v, error = %v", user, err)
						}
					} else if user.ID != 0 || !errors.Is(err, tt.denied) {
						t.Fatalf("disabled user = %#v, error = %v; want %v", user, err, tt.denied)
					}
				})
			}
			if !enabled {
				if _, _, err := local.SignIn(context.Background(), "example", "password"); !errors.Is(err, ErrInvalidCredentials) {
					t.Fatalf("disabled sign-in error = %v", err)
				}
				if _, err := local.ChangePassword(context.Background(), 7, "example", "current", "new"); !errors.Is(err, ErrInvalidCredentials) {
					t.Fatalf("disabled password-change error = %v", err)
				}
			}
		})
	}
}

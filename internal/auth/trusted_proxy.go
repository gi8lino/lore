package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/model"
)

// TrustedProxy authenticates requests using identity headers from a trusted proxy.
type TrustedProxy struct {
	// repository persists and resolves trusted-proxy users.
	repository trustedProxyRepository
	// headers maps trusted proxy headers to Lore identity fields.
	headers TrustedProxyHeaders
}

// TrustedProxyHeaders contains ordered header candidates for each external identity field.
type TrustedProxyHeaders struct {
	Username    []string
	Email       []string
	DisplayName []string
}

// NewTrustedProxy creates a trusted-proxy authenticator.
func NewTrustedProxy(repository trustedProxyRepository, headers TrustedProxyHeaders) *TrustedProxy {
	return &TrustedProxy{repository: repository, headers: headers}
}

// Authenticate resolves the first populated trusted identity header.
func (a *TrustedProxy) Authenticate(r *http.Request) (model.User, error) {
	username := firstHeader(r, a.headers.Username)
	if username == "" {
		return model.User{}, ErrUnauthenticated
	}

	user, err := a.repository.TrustedProxyUser(
		r.Context(),
		username,
		firstHeader(r, a.headers.Email),
		firstHeader(r, a.headers.DisplayName),
	)
	if errors.Is(err, model.ErrRegistrationDisabled) {
		return model.User{}, ErrRegistrationDisabled
	}
	return user, err
}

// firstHeader returns the first non-empty configured header value.
func firstHeader(r *http.Request, headers []string) string {
	for _, header := range headers {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			return value
		}
	}
	return ""
}

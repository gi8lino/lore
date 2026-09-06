package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
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
	Groups      []string
	AdminGroup  string
}

// NewTrustedProxy creates a trusted-proxy authenticator.
func NewTrustedProxy(repository trustedProxyRepository, headers TrustedProxyHeaders) *TrustedProxy {
	return &TrustedProxy{repository: repository, headers: headers}
}

// Authenticate resolves the first populated trusted identity header.
func (a *TrustedProxy) Authenticate(r *http.Request) (domain.User, error) {
	username := firstHeader(r, a.headers.Username)
	if username == "" {
		return domain.User{}, ErrUnauthenticated
	}

	user, err := a.repository.TrustedProxyUser(
		r.Context(),
		username,
		firstHeader(r, a.headers.Email),
		firstHeader(r, a.headers.DisplayName),
	)
	if errors.Is(err, domain.ErrRegistrationDisabled) {
		return domain.User{}, ErrRegistrationDisabled
	}
	if err != nil {
		return domain.User{}, err
	}
	if !user.Enabled {
		return domain.User{}, ErrInvalidCredentials
	}

	if a.headers.AdminGroup != "" {
		groups := splitHeaderValues(firstHeader(r, a.headers.Groups))
		externalAdmin := containsGroup(groups, a.headers.AdminGroup)
		if err := a.repository.SetExternalAdminStatus(r.Context(), user.ID, "trusted-proxy", externalAdmin); err != nil {
			return domain.User{}, err
		}

		if externalAdmin {
			user.ExternalAdmin = true
			user.Role = "admin"
		}
	}

	return user, err
}

// splitHeaderValues normalizes a comma-separated trusted group header.
func splitHeaderValues(value string) []string {
	parts := strings.Split(value, ",")
	groups := make([]string, 0, len(parts))

	for _, part := range parts {
		if group := strings.TrimSpace(part); group != "" {
			groups = append(groups, group)
		}
	}

	return groups
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

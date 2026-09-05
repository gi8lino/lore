package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/model"
)

// Bearer authenticates API requests using bearer tokens.
type Bearer struct {
	// repository resolves API tokens to wiki users.
	repository bearerRepository
}

// NewBearer creates a bearer-token authenticator.
func NewBearer(repository bearerRepository) *Bearer {
	return &Bearer{repository: repository}
}

// Authenticate resolves the bearer token in the Authorization header.
func (a *Bearer) Authenticate(r *http.Request) (model.User, error) {
	value, supplied := r.Header["Authorization"]
	if !supplied {
		return model.User{}, ErrUnauthenticated
	}

	token, ok := parseBearerValue(strings.Join(value, ","))
	if !ok {
		return model.User{}, ErrInvalidCredentials
	}

	user, err := a.repository.UserByToken(r.Context(), token)
	if errors.Is(err, model.ErrNotFound) {
		return model.User{}, ErrInvalidCredentials
	}

	return user, err
}

// parseBearerValue validates an explicit Authorization header and returns its bearer token.
func parseBearerValue(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(fields[1])

	return token, token != ""
}

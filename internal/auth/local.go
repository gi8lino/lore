package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const (
	localSessionCookie = "lore_local_session"
	localSessionTTL    = 12 * time.Hour
	minimumPasswordLen = 12
)

// Local authenticates optional password-backed Lore accounts.
type Local struct {
	repository localRepository
	publicURL  string
}

// NewLocal creates the local-login authenticator used by setup and recovery login.
func NewLocal(repository localRepository, publicURL string) *Local {
	return &Local{repository: repository, publicURL: strings.TrimSpace(publicURL)}
}

// Authenticate resolves a valid local session cookie.
func (l *Local) Authenticate(r *http.Request) (model.User, error) {
	cookie, err := r.Cookie(localSessionCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return model.User{}, ErrUnauthenticated
	}

	user, err := l.repository.LocalUserBySession(r.Context(), localSessionHash(cookie.Value))
	if errors.Is(err, model.ErrNotFound) {
		return model.User{}, ErrUnauthenticated
	}

	return user, err
}

// SignIn verifies local credentials and creates a new browser session.
func (l *Local) SignIn(ctx context.Context, username, password string) (model.User, string, error) {
	user, passwordHash, err := l.repository.LocalCredential(ctx, strings.TrimSpace(username))
	if errors.Is(err, model.ErrNotFound) {
		return model.User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return model.User{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return model.User{}, "", ErrInvalidCredentials
	}

	token, err := newLocalSessionToken()
	if err != nil {
		return model.User{}, "", err
	}
	if err := l.repository.CreateLocalSession(ctx, user.ID, localSessionHash(token), time.Now().Add(localSessionTTL)); err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

// ChangePassword verifies the current credential, replaces it, and creates a fresh session.
func (l *Local) ChangePassword(
	ctx context.Context,
	userID int64,
	username, currentPassword, newPassword string,
) (string, error) {
	user, passwordHash, err := l.repository.LocalCredential(ctx, strings.TrimSpace(username))
	if errors.Is(err, model.ErrNotFound) || (err == nil && user.ID != userID) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)) != nil {
		return "", ErrInvalidCredentials
	}

	newHash, err := localPasswordHash(newPassword)
	if err != nil {
		return "", err
	}
	if err := l.repository.SetLocalCredential(ctx, userID, newHash); err != nil {
		return "", err
	}

	token, err := newLocalSessionToken()
	if err != nil {
		return "", err
	}
	if err := l.repository.CreateLocalSession(ctx, userID, localSessionHash(token), time.Now().Add(localSessionTTL)); err != nil {
		return "", err
	}

	return token, nil
}

// Setup creates the first local administrator and starts its initial session.
func (l *Local) Setup(
	ctx context.Context,
	username, email, displayName, password string,
) (model.User, string, error) {
	passwordHash, err := localPasswordHash(password)
	if err != nil {
		return model.User{}, "", err
	}

	user, err := l.repository.CreateInitialLocalAdministrator(ctx, username, email, displayName, passwordHash)
	if err != nil {
		return model.User{}, "", err
	}

	token, err := newLocalSessionToken()
	if err != nil {
		return model.User{}, "", err
	}
	if err := l.repository.CreateLocalSession(ctx, user.ID, localSessionHash(token), time.Now().Add(localSessionTTL)); err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

// SetPassword creates or replaces one Lore user's local recovery password.
func (l *Local) SetPassword(ctx context.Context, userID int64, password string) error {
	passwordHash, err := localPasswordHash(password)
	if err != nil {
		return err
	}

	return l.repository.SetLocalCredential(ctx, userID, passwordHash)
}

// WriteSessionCookie stores a local session token in an HTTP-only cookie.
func (l *Local) WriteSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     localSessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(localSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   strings.HasPrefix(l.publicURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSession revokes the current local session and removes its cookie.
func (l *Local) ClearSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(localSessionCookie); err == nil && cookie.Value != "" {
		_ = l.repository.DeleteLocalSession(r.Context(), localSessionHash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     localSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   strings.HasPrefix(l.publicURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidLocalPassword reports whether a password satisfies the local-login minimum length.
func ValidLocalPassword(password string) bool {
	return len([]rune(password)) >= minimumPasswordLen
}

// localPasswordHash hashes a validated local password with bcrypt.
func localPasswordHash(password string) (string, error) {
	if !ValidLocalPassword(password) {
		return "", errors.New("local password must be at least 12 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(hash), err
}

// newLocalSessionToken creates an opaque random browser-session token.
func newLocalSessionToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

// localSessionHash returns the at-rest representation of a local session token.
func localSessionHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

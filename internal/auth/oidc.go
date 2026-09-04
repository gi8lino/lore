package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/model"
	"golang.org/x/oauth2"
)

// OIDC authenticates browser sessions and handles the OIDC authorization flow.
type OIDC struct {
	// repository persists and resolves authenticated OIDC users.
	repository oidcRepository
	// verifier validates OIDC identity tokens.
	verifier *oidc.IDTokenVerifier
	// oauth contains the OIDC authorization-code client configuration.
	oauth *oauth2.Config
	// secret signs and verifies application cookies.
	secret []byte
	// issuer is the verified provider namespace used with OIDC subjects.
	issuer string
	// publicURL is the externally visible wiki base URL.
	publicURL string
	// groupClaim is the top-level ID-token claim containing external groups.
	groupClaim string
	// groupSync enables mapped group membership synchronization at login.
	groupSync bool
	// groupsAuthoritative removes mapped memberships absent from the current claim.
	groupsAuthoritative bool
	// groupMappings maps external claim values to Lore groups.
	groupMappings []model.OIDCGroupMapping
}

// claims contains the OIDC identity claims used to create a wiki user.
type claims struct {
	// Email is the user email returned by the identity provider.
	Email string `json:"email"`
	// PreferredUsername is the preferred login name returned by the identity provider.
	PreferredUsername string `json:"preferred_username"`
	// Name is the display name returned by the identity provider.
	Name string `json:"name"`
}

// session contains only the stable OIDC identity needed to resolve a Lore user.
type session struct {
	// Issuer namespaces the OIDC subject.
	Issuer string `json:"iss"`
	// Subject is the stable user identifier assigned by the issuer.
	Subject string `json:"sub"`
	// Expires is the Unix timestamp after which the session is invalid.
	Expires int64 `json:"x"`
}

// loginState contains the signed OIDC state and its expiration.
type loginState struct {
	// State is the saved OIDC state value.
	State string `json:"s"`
	// Expires is the Unix expiration timestamp for the saved state.
	Expires int64 `json:"x"`
}

// nextLocation contains the local path to restore after authentication.
type nextLocation struct {
	// Path is the local request path to restore after authentication.
	Path string `json:"n"`
}

// NewOIDC creates an OIDC authenticator and authorization-flow handler.
func NewOIDC(
	ctx context.Context,
	config OIDCConfig,
	repository oidcRepository,
) (*OIDC, error) {
	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, err
	}

	return &OIDC{
		repository:          repository,
		verifier:            provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		secret:              []byte(config.SessionSecret),
		issuer:              strings.TrimSpace(config.Issuer),
		publicURL:           config.PublicURL,
		groupClaim:          strings.TrimSpace(config.GroupClaim),
		groupSync:           config.GroupSync,
		groupsAuthoritative: config.GroupsAuthoritative,
		groupMappings:       append([]model.OIDCGroupMapping(nil), config.GroupMappings...),
		oauth: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  strings.TrimRight(config.PublicURL, "/") + "/auth/callback",
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}, nil
}

// Authenticate resolves the authenticated OIDC browser session by issuer and subject.
func (o *OIDC) Authenticate(r *http.Request) (model.User, error) {
	var current session
	if err := o.decodeCookie(r, "lore_session", &current); err != nil {
		return model.User{}, ErrUnauthenticated
	}
	if current.Expires < time.Now().Unix() || current.Issuer != o.issuer || current.Subject == "" {
		return model.User{}, ErrUnauthenticated
	}

	user, err := o.repository.OIDCUser(r.Context(), current.Issuer, current.Subject)
	if errors.Is(err, model.ErrNotFound) {
		return model.User{}, ErrUnauthenticated
	}
	return user, err
}

// Login starts the OIDC authorization-code flow.
func (o *OIDC) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stateBytes := make([]byte, 24)
		if _, err := rand.Read(stateBytes); err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}

		state := base64.RawURLEncoding.EncodeToString(stateBytes)
		o.setCookie(w, "lore_state", loginState{
			State:   state,
			Expires: time.Now().Add(10 * time.Minute).Unix(),
		}, 600)

		if next := r.URL.Query().Get("next"); isLocalPath(next) {
			o.setCookie(w, "lore_next", nextLocation{Path: next}, 600)
		}

		http.Redirect(w, r, o.oauth.AuthCodeURL(state), http.StatusFound)
	}
}

// Callback completes the OIDC flow and establishes the browser session.
func (o *OIDC) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var saved loginState
		if o.decodeCookie(r, "lore_state", &saved) != nil || saved.State != r.URL.Query().Get("state") ||
			saved.Expires < time.Now().Unix() {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid login state.")
			return
		}

		token, err := o.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			httpresponse.Problem(w, http.StatusUnauthorized, "Login failed.")
			return
		}

		raw, _ := token.Extra("id_token").(string)
		idToken, err := o.verifier.Verify(r.Context(), raw)
		if err != nil {
			httpresponse.Problem(w, http.StatusUnauthorized, "Invalid identity.")
			return
		}

		issuer := strings.TrimSpace(idToken.Issuer)
		subject := strings.TrimSpace(idToken.Subject)
		if issuer == "" || issuer != o.issuer || subject == "" {
			httpresponse.Problem(w, http.StatusUnauthorized, "Invalid identity.")
			return
		}

		var identity claims
		if err := idToken.Claims(&identity); err != nil {
			httpresponse.Problem(w, http.StatusUnauthorized, "Invalid claims.")
			return
		}

		username := strings.TrimSpace(identity.PreferredUsername)
		if username == "" {
			httpresponse.Problem(w, http.StatusUnauthorized, "The preferred_username claim is required.")
			return
		}

		var groups []string
		if o.groupSync {
			groups, err = oidcGroups(idToken, o.groupClaim)
			if err != nil {
				httpresponse.Problem(w, http.StatusUnauthorized, "Invalid group claim.")
				return
			}
		}

		user, err := o.repository.LoginOIDCUser(
			r.Context(),
			issuer,
			subject,
			username,
			identity.Email,
			identity.Name,
		)
		if err != nil {
			switch {
			case errors.Is(err, model.ErrIdentityApprovalRequired):
				httpresponse.Problem(w, http.StatusForbidden, "Registration is closed. Your verified identity is awaiting administrator approval.")
			case errors.Is(err, model.ErrIdentityRejected):
				httpresponse.Problem(w, http.StatusForbidden, "This identity has been rejected by an administrator.")
			case errors.Is(err, model.ErrRegistrationDisabled):
				httpresponse.Problem(w, http.StatusForbidden, "User registration is disabled.")
			case errors.Is(err, model.ErrAlreadyExists):
				httpresponse.Problem(w, http.StatusConflict, "The preferred username is already used by another account.")
			default:
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			}
			return
		}

		if o.groupSync {
			if err := o.repository.SyncOIDCGroups(
				r.Context(),
				user.ID,
				groups,
				o.groupMappings,
				o.groupsAuthoritative,
			); err != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
		}

		o.setCookie(w, "lore_session", session{
			Issuer:  issuer,
			Subject: subject,
			Expires: time.Now().Add(12 * time.Hour).Unix(),
		}, 43200)

		next := "/"
		var savedNext nextLocation
		if o.decodeCookie(r, "lore_next", &savedNext) == nil && isLocalPath(savedNext.Path) {
			next = savedNext.Path
		}
		http.Redirect(w, r, next, http.StatusFound)
	}
}

// oidcGroups extracts a configurable top-level string or string-array group claim.
func oidcGroups(idToken *oidc.IDToken, claim string) ([]string, error) {
	claim = strings.TrimSpace(claim)
	if claim == "" {
		return nil, nil
	}

	var values map[string]json.RawMessage
	if err := idToken.Claims(&values); err != nil {
		return nil, err
	}
	return oidcGroupValues(values[claim])
}

// oidcGroupValues normalizes one optional OIDC group claim.
func oidcGroupValues(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var groups []string
	if err := json.Unmarshal(raw, &groups); err != nil {
		var group string
		if stringErr := json.Unmarshal(raw, &group); stringErr != nil {
			return nil, errors.New("OIDC group claim must be a string or string array")
		}
		groups = []string{group}
	}

	seen := make(map[string]bool, len(groups))
	normalized := groups[:0]
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" || seen[group] {
			continue
		}
		seen[group] = true
		normalized = append(normalized, group)
	}
	return normalized, nil
}

// Logout clears OIDC and local browser sessions and redirects home.
func Logout(local *Local) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if local != nil {
			local.ClearSession(w, r)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "lore_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// LoginUnavailable redirects to the home page when the configured auth mode has no login flow.
func LoginUnavailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// setCookie encodes, signs, and writes a short-lived application cookie.
func (o *OIDC) setCookie(w http.ResponseWriter, name string, value any, maxAge int) {
	data, _ := json.Marshal(value)
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, o.secret)
	_, _ = mac.Write([]byte(payload))
	signed := payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    signed,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   strings.HasPrefix(o.publicURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
}

// decodeCookie verifies and decodes a signed application cookie.
func (o *OIDC) decodeCookie(r *http.Request, name string, out any) error {
	if len(o.secret) == 0 {
		return errors.New("no secret")
	}

	cookie, err := r.Cookie(name)
	if err != nil {
		return err
	}

	payload, signature, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return errors.New("invalid cookie")
	}

	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, o.secret)
	_, _ = mac.Write([]byte(payload))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return errors.New("invalid signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode cookie: %w", err)
	}
	return nil
}

// isLocalPath reports whether a redirect target stays within this application.
func isLocalPath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}

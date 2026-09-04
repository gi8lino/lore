package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
)

// LocalLogin renders and processes the optional Lore-managed sign-in flow.
func LocalLogin(
	settingsUseCases settingsService,
	systemUseCases systemService,
	browserAuth auth.BrowserAuth,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := browserAuth.LocalLoginAllowed(r.Context())
		if err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}
		if !allowed {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}

		if views.runtime.AuthModeOverride == "" {
			settings, err := settingsUseCases.ApplicationSettings(r.Context())
			if err != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			required, err := systemUseCases.SetupRequired(r.Context())
			if err != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			if required && settings.Authentication.Mode == string(auth.AuthModeNone) {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
		}

		next := safeAuthNext(r.URL.Query().Get("next"))
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				httpresponse.Problem(w, http.StatusBadRequest, "Invalid login form.")
				return
			}
			next = safeAuthNext(r.FormValue("next"))
			_, token, err := browserAuth.Local.SignIn(r.Context(), r.FormValue("username"), r.FormValue("password"))
			if err == nil {
				browserAuth.Local.WriteSessionCookie(w, token)
				if next == "" {
					next = "/"
				}
				http.Redirect(w, r, next, http.StatusSeeOther)
				return
			}
			if !errors.Is(err, auth.ErrInvalidCredentials) {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			data, dataErr := publicViewData(views, "Local sign in")
			if dataErr != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			data.AuthError = "Invalid username or password."
			data.AuthNext = next
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			renderPublic(views, w, "login", data)
			return
		}

		data, err := publicViewData(views, "Local sign in")
		if err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}
		data.AuthNext = next
		renderPublic(views, w, "login", data)
	}
}

// Setup renders and processes the one-time first-administrator bootstrap.
func Setup(
	settingsUseCases settingsService,
	systemUseCases systemService,
	browserAuth auth.BrowserAuth,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// An explicit runtime authentication override is itself a bootstrap or
		// recovery choice, so do not expose the unauthenticated setup surface.
		if views.runtime.AuthModeOverride != "" {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}
		settings, err := settingsUseCases.ApplicationSettings(r.Context())
		if err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}
		if settings.Authentication.Mode != string(auth.AuthModeNone) {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}
		required, err := systemUseCases.SetupRequired(r.Context())
		if err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}
		if !required {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				httpresponse.Problem(w, http.StatusBadRequest, "Invalid setup form.")
				return
			}
			errorMessage := setupValidationError(r)
			if errorMessage == "" {
				user, token, err := browserAuth.Local.Setup(
					r.Context(),
					r.FormValue("username"),
					r.FormValue("email"),
					r.FormValue("display_name"),
					r.FormValue("password"),
				)
				if err == nil {
					browserAuth.Local.WriteSessionCookie(w, token)
					systemUseCases.RecordSetupCompleted(r.Context(), user)
					http.Redirect(w, r, "/admin/configuration", http.StatusSeeOther)
					return
				}
				if errors.Is(err, service.ErrAlreadyExists) || errors.Is(err, service.ErrForbidden) {
					httpresponse.Problem(w, http.StatusNotFound, "Not found.")
					return
				}
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}

			data, dataErr := publicViewData(views, "Set up Lore")
			if dataErr != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			data.AuthError = errorMessage
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			renderPublic(views, w, "setup", data)
			return
		}

		data, err := publicViewData(views, "Set up Lore")
		if err != nil {
			httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
			return
		}
		renderPublic(views, w, "setup", data)
	}
}

// setupValidationError returns the first compact validation error for first-run setup.
func setupValidationError(r *http.Request) string {
	if strings.TrimSpace(r.FormValue("username")) == "" {
		return "Username is required."
	}
	password := r.FormValue("password")
	if !auth.ValidLocalPassword(password) {
		return "Password must contain at least 12 characters."
	}
	if password != r.FormValue("password_confirm") {
		return "Passwords do not match."
	}
	return ""
}

// safeAuthNext accepts only local paths as post-authentication destinations.
func safeAuthNext(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return ""
	}
	return value
}

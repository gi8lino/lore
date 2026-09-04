package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
)

// CreatePersonalToken issues a personal access token for the current user.
func CreatePersonalToken(tokenUseCases tokenService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w,
				http.StatusUnauthorized,
				"Unauthorized.",
				httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
			)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		name, expiresAt, err := parseTokenForm(r)
		if err != nil {
			writeTokenFormError(w, err)
			return
		}
		issued, err := tokenUseCases.CreateToken(r.Context(), name, user.ID, user.ID, expiresAt)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusCreated, issued)
	}
}

// DeletePersonalToken revokes a personal access token owned by the current user.
func DeletePersonalToken(tokenUseCases tokenService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w,
				http.StatusUnauthorized,
				"Unauthorized.",
				httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
			)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid token identifier.")
			return
		}
		if err := tokenUseCases.DeleteUserToken(r.Context(), id, user.ID); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				httpresponse.Problem(w, http.StatusNotFound, "Token not found.")
				return
			}
			writeUnexpectedProblem(logger, w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// CreateAdminToken issues a personal access token for any selected account.
func CreateAdminToken(tokenUseCases tokenService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := currentUser(r)

		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		name, expiresAt, err := parseTokenForm(r)
		if err != nil {
			writeTokenFormError(w, err)
			return
		}
		userID, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
		if err != nil || userID <= 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Token validation failed.",
				httpresponse.NewFieldProblem("user_id", "Choose a valid user."),
			)
			return
		}
		issued, err := tokenUseCases.CreateToken(r.Context(), name, userID, admin.ID, expiresAt)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				httpresponse.Problem(w, http.StatusNotFound, "User not found.")
				return
			}
			writeUnexpectedProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusCreated, issued)
	}
}

// DeleteAdminToken revokes any personal access token.
func DeleteAdminToken(tokenUseCases tokenService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid token identifier.")
			return
		}
		if err := tokenUseCases.DeleteToken(r.Context(), id); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				httpresponse.Problem(w, http.StatusNotFound, "Token not found.")
				return
			}
			writeUnexpectedProblem(logger, w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// tokenFormError describes one invalid shared token form field.
type tokenFormError struct {
	field   string
	message string
}

// Error returns the user-facing token form validation message.
func (e *tokenFormError) Error() string {
	return e.message
}

// parseTokenForm validates the shared token name and optional expiration date fields.
func parseTokenForm(r *http.Request) (string, *time.Time, error) {
	if err := r.ParseForm(); err != nil {
		return "", nil, err
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return "", nil, &tokenFormError{field: "name", message: "Token name is required."}
	}
	if len(name) > 120 {
		return "", nil, &tokenFormError{field: "name", message: "Token name is too long."}
	}

	value := strings.TrimSpace(r.FormValue("expires"))
	if value == "" {
		return name, nil, nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", nil, &tokenFormError{field: "expires", message: "Enter a valid expiration date."}
	}
	expiresAt := date.AddDate(0, 0, 1).Add(-time.Nanosecond)
	if !expiresAt.After(time.Now().UTC()) {
		return "", nil, &tokenFormError{field: "expires", message: "Expiration date must be in the future."}
	}
	return name, &expiresAt, nil
}

// writeTokenFormError writes browser-facing token form parsing and validation errors.
func writeTokenFormError(w http.ResponseWriter, err error) {
	if validation, ok := errors.AsType[*tokenFormError](err); ok {
		httpresponse.Problem(w,
			http.StatusBadRequest,
			"Token validation failed.",
			httpresponse.NewFieldProblem(validation.field, validation.message),
		)
		return
	}
	httpresponse.Problem(w, http.StatusBadRequest, "Invalid token form.")
}

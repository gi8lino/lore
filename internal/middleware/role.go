package middleware

import (
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
)

// RequireRole authorizes an authenticated request against one of the allowed roles.
func RequireRole(roles ...string) Middleware {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.User(r)
			if !ok {
				roleUnauthorized(w, r)
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				roleForbidden(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// roleUnauthorized writes a protocol-appropriate response when authentication middleware was omitted.
func roleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		httpresponse.Problem(w,
			http.StatusUnauthorized,
			"Unauthorized.",
			httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
		)
		return
	}
	httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
}

// roleForbidden writes a protocol-appropriate response when the current role is not allowed.
func roleForbidden(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		httpresponse.Problem(w,
			http.StatusForbidden,
			"Forbidden.",
			httpresponse.NewFieldProblem(
				"authorization",
				"Your account does not have permission to perform this action.",
			),
		)
		return
	}
	httpresponse.Problem(w, http.StatusForbidden, "Forbidden.")
}

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
)

type pageAccessPolicy interface {
	CanView(context.Context, domain.User, string) (bool, error)
	CanEdit(context.Context, domain.User, string) (bool, error)
}

// RequirePageView hides page routes denied by inherited path access rules.
func RequirePageView(access pageAccessPolicy) Middleware {
	return requirePageAccess(access, false)
}

// RequirePageEdit rejects mutations denied by inherited path access rules.
func RequirePageEdit(access pageAccessPolicy) Middleware {
	return requirePageAccess(access, true)
}

func requirePageAccess(access pageAccessPolicy, edit bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.User(r)
			if !ok {
				roleUnauthorized(w, r)
				return
			}
			path := strings.TrimSpace(r.PathValue("slug"))
			if path == "" {
				next.ServeHTTP(w, r)
				return
			}
			allowed, err := access.CanView(r.Context(), user, path)
			if edit {
				allowed, err = access.CanEdit(r.Context(), user, path)
			}
			if err != nil {
				httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				return
			}
			if !allowed {
				if edit {
					roleForbidden(w, r)
				} else {
					httpresponse.Problem(w, http.StatusNotFound, "Page not found.")
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

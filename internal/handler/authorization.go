package handler

import (
	"net/http"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/service"
)

// currentUser returns the user populated by route authentication middleware.
func currentUser(r *http.Request) service.User {
	user, _ := auth.User(r)
	return user
}

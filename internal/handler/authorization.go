package handler

import (
	"net/http"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/model"
)

// currentUser returns the user populated by route authentication middleware.
func currentUser(r *http.Request) model.User {
	user, _ := auth.User(r)
	return user
}

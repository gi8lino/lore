package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
)

type robotsSettingsService interface {
	ApplicationSettings(context.Context) (domain.ApplicationSettings, error)
}

// Robots serves crawler guidance configured by an administrator.
func Robots(settingsUseCases robotsSettingsService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := settingsUseCases.ApplicationSettings(r.Context())
		if err != nil {
			httpresponse.InternalServerError(logger, w, err)
			return
		}

		var body string

		switch settings.RobotsPolicy {
		case domain.RobotsPolicyAllow:
			body = "User-agent: *\nAllow: /\n"
		case domain.RobotsPolicyDisallow:
			body = "User-agent: *\nDisallow: /\n"
		case domain.RobotsPolicyNone:
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		default:
			httpresponse.InternalServerError(logger, w, errors.New("invalid persisted robots.txt policy"))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

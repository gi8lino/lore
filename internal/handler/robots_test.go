package handler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
)

type robotsSettingsStub struct {
	settings domain.ApplicationSettings
	err      error
}

func (s robotsSettingsStub) ApplicationSettings(context.Context) (domain.ApplicationSettings, error) {
	return s.settings, s.err
}

func TestRobots(t *testing.T) {
	t.Parallel()

	t.Run("allows indexing", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		handler := Robots(
			robotsSettingsStub{settings: domain.ApplicationSettings{RobotsPolicy: domain.RobotsPolicyAllow}},
			slog.Default(),
		)

		handler.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "text/plain; charset=utf-8", response.Header().Get("Content-Type"))
		assert.Equal(t, "User-agent: *\nAllow: /\n", response.Body.String())
	})

	t.Run("disallows indexing", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		handler := Robots(
			robotsSettingsStub{settings: domain.ApplicationSettings{RobotsPolicy: domain.RobotsPolicyDisallow}},
			slog.Default(),
		)

		handler.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "User-agent: *\nDisallow: /\n", response.Body.String())
	})

	t.Run("can be disabled", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		handler := Robots(
			robotsSettingsStub{settings: domain.ApplicationSettings{RobotsPolicy: domain.RobotsPolicyNone}},
			slog.Default(),
		)

		handler.ServeHTTP(response, request)

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("reports settings failures", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
		handler := Robots(robotsSettingsStub{err: errors.New("database unavailable")}, logger)

		handler.ServeHTTP(response, request)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, logs.String(), "database unavailable")
	})
}

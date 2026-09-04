package service

import (
	"context"
)

// settingsRepository contains persisted application configuration operations.
type settingsRepository interface {
	auditRepository
	ApplicationSettings(context.Context) (ApplicationSettings, error)
	SaveApplicationSettings(context.Context, ApplicationSettings) error
	SaveAuthenticationSettings(context.Context, AuthenticationSettings) error
	SaveRenderingSettings(context.Context, RenderingSettings) error
}

// Settings exposes persisted application configuration use cases.
type Settings struct{ repository settingsRepository }

// NewSettings constructs the application settings service.
func NewSettings(repository settingsRepository) *Settings { return &Settings{repository: repository} }

// ApplicationSettings returns the current application configuration.
func (s *Settings) ApplicationSettings(ctx context.Context) (ApplicationSettings, error) {
	return s.repository.ApplicationSettings(ctx)
}

// SaveApplicationSettings persists application settings and records the change.
func (s *Settings) SaveApplicationSettings(
	ctx context.Context,
	settings ApplicationSettings,
	actorID int64,
) error {
	if err := s.repository.SaveApplicationSettings(ctx, settings); err != nil {
		return err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"settings.application_updated",
		"settings",
		"application",
		"Updated application settings",
	)
	return nil
}

// SaveAuthenticationSettings persists authentication settings and records the change.
func (s *Settings) SaveAuthenticationSettings(
	ctx context.Context,
	settings AuthenticationSettings,
	actorID int64,
) error {
	if err := s.repository.SaveAuthenticationSettings(ctx, settings); err != nil {
		return err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"settings.authentication_updated",
		"settings",
		"authentication",
		"Updated authentication settings",
	)
	return nil
}

// SaveRenderingSettings persists rendering settings and records the change.
func (s *Settings) SaveRenderingSettings(
	ctx context.Context,
	settings RenderingSettings,
	actorID int64,
) error {
	if err := s.repository.SaveRenderingSettings(ctx, settings); err != nil {
		return err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"settings.rendering_updated",
		"settings",
		"rendering",
		"Updated rendering settings",
	)
	return nil
}

// RecordLocalPasswordUpdated records a local recovery password change.
func (s *Settings) RecordLocalPasswordUpdated(ctx context.Context, actor User) {
	_ = audit(s.repository,
		ctx,
		actor.ID,
		"settings.local_password_updated",
		"user",
		actor.Username,
		"Updated local recovery password",
	)
}

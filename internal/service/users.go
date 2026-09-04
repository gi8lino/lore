package service

import (
	"context"
	"fmt"
)

// userRepository contains account and external identity administration operations.
type userRepository interface {
	auditRepository
	Users(context.Context) ([]AdminUser, error)
	User(context.Context, int64) (User, error)
	UserGroups(context.Context, int64) ([]Group, error)
	UpdateUser(context.Context, int64, string, []int64) error
	SearchUsers(context.Context, string, int) ([]User, error)
	OIDCIdentities(context.Context) ([]OIDCIdentity, error)
	OIDCGroupMappings(context.Context) ([]OIDCGroupMapping, error)
	PendingOIDCIdentities(context.Context) ([]PendingOIDCIdentity, error)
	ApprovePendingOIDCIdentity(context.Context, int64) (User, error)
	LinkPendingOIDCIdentity(context.Context, int64, int64) (User, error)
	SetPendingOIDCIdentityRejected(context.Context, int64, bool) error
	RemoveOIDCIdentity(context.Context, int64, string, string) error
	HasLocalCredential(context.Context, int64) (bool, error)
}

// Users exposes account and external identity administration use cases.
type Users struct{ repository userRepository }

// NewUsers constructs the account administration service.
func NewUsers(repository userRepository) *Users { return &Users{repository: repository} }

// Users returns accounts with administration metadata.
func (s *Users) Users(ctx context.Context) ([]AdminUser, error) {
	return s.repository.Users(ctx)
}

// User returns an account by identifier.
func (s *Users) User(ctx context.Context, id int64) (User, error) {
	return s.repository.User(ctx, id)
}

// UserGroups returns the groups assigned to a user.
func (s *Users) UserGroups(ctx context.Context, userID int64) ([]Group, error) {
	return s.repository.UserGroups(ctx, userID)
}

// UpdateUser replaces a user's role and group assignments.
func (s *Users) UpdateUser(ctx context.Context, userID int64, role string, groupIDs []int64) error {
	return s.repository.UpdateUser(ctx, userID, role, groupIDs)
}

// SearchUsers returns accounts matching a display or login query.
func (s *Users) SearchUsers(ctx context.Context, query string, limit int) ([]User, error) {
	return s.repository.SearchUsers(ctx, query, limit)
}

// OIDCIdentities returns the external identities linked to user accounts.
func (s *Users) OIDCIdentities(ctx context.Context) ([]OIDCIdentity, error) {
	return s.repository.OIDCIdentities(ctx)
}

// OIDCGroupMappings returns external-to-local group mappings.
func (s *Users) OIDCGroupMappings(ctx context.Context) ([]OIDCGroupMapping, error) {
	return s.repository.OIDCGroupMappings(ctx)
}

// PendingOIDCIdentities returns external identities awaiting administrator action.
func (s *Users) PendingOIDCIdentities(ctx context.Context) ([]PendingOIDCIdentity, error) {
	return s.repository.PendingOIDCIdentities(ctx)
}

// ApprovePendingOIDCIdentity creates a user from a pending identity and audits it.
func (s *Users) ApprovePendingOIDCIdentity(
	ctx context.Context,
	pendingID, actorID int64,
) (User, error) {
	user, err := s.repository.ApprovePendingOIDCIdentity(ctx, pendingID)
	if err != nil {
		return User{}, err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"identity.oidc_approved",
		"user",
		fmt.Sprint(user.ID),
		"Created user from pending OIDC identity",
	)
	return user, nil
}

// LinkPendingOIDCIdentity binds a pending identity to a user and audits it.
func (s *Users) LinkPendingOIDCIdentity(
	ctx context.Context,
	pendingID, userID int64,
	actorID int64,
) (User, error) {
	user, err := s.repository.LinkPendingOIDCIdentity(ctx, pendingID, userID)
	if err != nil {
		return User{}, err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"identity.oidc_linked",
		"user",
		fmt.Sprint(user.ID),
		"Replaced OIDC identity binding for "+user.Username,
	)
	return user, nil
}

// SetPendingOIDCIdentityRejected rejects or reopens a pending identity and audits it.
func (s *Users) SetPendingOIDCIdentityRejected(
	ctx context.Context,
	pendingID int64,
	rejected bool,
	actorID int64,
) error {
	if err := s.repository.SetPendingOIDCIdentityRejected(ctx, pendingID, rejected); err != nil {
		return err
	}
	action := "identity.oidc_reopened"
	detail := "Reopened pending OIDC identity"
	if rejected {
		action = "identity.oidc_rejected"
		detail = "Rejected pending OIDC identity"
	}
	_ = audit(s.repository, ctx, actorID, action, "oidc_identity", fmt.Sprint(pendingID), detail)
	return nil
}

// RemoveOIDCIdentity disconnects an external identity and records the change.
func (s *Users) RemoveOIDCIdentity(
	ctx context.Context,
	userID int64,
	issuer, subject string,
	actorID int64,
) error {
	if err := s.repository.RemoveOIDCIdentity(ctx, userID, issuer, subject); err != nil {
		return err
	}
	_ = audit(s.repository,
		ctx,
		actorID,
		"identity.oidc_removed",
		"user",
		fmt.Sprint(userID),
		"Disconnected OIDC identity",
	)
	return nil
}

// HasLocalCredential reports whether a user has a local password.
func (s *Users) HasLocalCredential(ctx context.Context, userID int64) (bool, error) {
	return s.repository.HasLocalCredential(ctx, userID)
}

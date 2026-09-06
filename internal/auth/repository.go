package auth

import (
	"context"
	"time"

	"github.com/gi8lino/lore/internal/model"
)

// bearerRepository resolves API tokens to users.
type bearerRepository interface {
	UserByToken(context.Context, string) (model.User, error)
}

// noneRepository provisions the fixed administrator used by no-auth mode.
type noneRepository interface {
	EnsureAdministrator(context.Context, string, string, string) (model.User, error)
}

// trustedProxyRepository resolves identities asserted by a trusted proxy.
type trustedProxyRepository interface {
	TrustedProxyUser(context.Context, string, string, string) (model.User, error)
	SetExternalAdminStatus(context.Context, int64, string, bool) error
}

// localRepository persists local credentials and browser sessions.
type localRepository interface {
	CreateInitialLocalAdministrator(context.Context, string, string, string, string) (model.User, error)
	CreateLocalSession(context.Context, int64, string, time.Time) error
	DeleteLocalSession(context.Context, string) error
	LocalCredential(context.Context, string) (user model.User, passwordHash string, err error)
	LocalUserBySession(context.Context, string) (model.User, error)
	SetLocalCredential(context.Context, int64, string) error
}

// oidcRepository persists OIDC identities and synchronized group memberships.
type oidcRepository interface {
	LoginOIDCUser(context.Context, string, string, string, string, string) (model.User, error)
	OIDCUser(context.Context, string, string) (model.User, error)
	SyncOIDCGroups(context.Context, int64, []string, []model.OIDCGroupMapping, bool) error
	SetExternalAdminStatus(context.Context, int64, string, bool) error
}

// browserRepository collects the persistence capabilities used by dynamic browser authentication.
type browserRepository interface {
	localRepository
	noneRepository
	oidcRepository
	trustedProxyRepository
	ApplicationSettings(context.Context) (model.ApplicationSettings, error)
	HasLocalAdministratorCredential(context.Context) (bool, error)
	OIDCGroupMappings(context.Context) ([]model.OIDCGroupMapping, error)
	SetupRequired(context.Context) (bool, error)
}

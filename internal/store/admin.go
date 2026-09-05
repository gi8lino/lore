package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Stats returns high-level database counts for administrators.
func (s *Store) Stats(ctx context.Context) (AdminStats, error) {
	var stats AdminStats
	err := s.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM users),
  (SELECT count(*) FROM wiki_groups),
  (SELECT count(*) FROM pages WHERE deleted_at IS NULL),
  (SELECT count(*) FROM pages WHERE deleted_at IS NOT NULL),
  (SELECT count(*) FROM tags),
  (SELECT count(*) FROM images),
  (SELECT count(*) FROM api_tokens)`).Scan(
		&stats.Users,
		&stats.Groups,
		&stats.Pages,
		&stats.DeletedPages,
		&stats.Tags,
		&stats.Images,
		&stats.Tokens,
	)
	return stats, err
}

// Users returns all wiki users with their group memberships.
func (s *Store) Users(ctx context.Context) ([]AdminUser, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
  u.id,
  u.username,
  u.email,
  u.display_name,
  u.role,
  u.enabled,
  (u.oidc_admin_observed OR u.trusted_proxy_admin_observed),
  (u.oidc_external_admin OR u.trusted_proxy_external_admin),
  lc.user_id IS NOT NULL,
  coalesce(lc.enabled,false),
  coalesce(u.last_login, u.created_at),
  u.last_login IS NOT NULL,
  coalesce(array_agg(g.name ORDER BY g.name) FILTER (WHERE g.id IS NOT NULL),'{}')
FROM users u
LEFT JOIN local_credentials lc ON lc.user_id=u.id
LEFT JOIN user_groups ug ON ug.user_id=u.id
LEFT JOIN wiki_groups g ON g.id=ug.group_id
GROUP BY u.id,lc.user_id,lc.enabled
ORDER BY lower(u.display_name),lower(u.username),u.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(
			&user.User.ID,
			&user.User.Username,
			&user.User.Email,
			&user.User.DisplayName,
			&user.User.Role,
			&user.User.Enabled,
			&user.ExternalAdminObserved,
			&user.ExternalAdmin,
			&user.HasLocalCredential,
			&user.LocalCredentialEnabled,
			&user.LastLogin,
			&user.HasLoggedIn,
			&user.Groups,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// UpdateUser updates an account role, enabled state, local-login state, and group memberships transactionally.
func (s *Store) UpdateUser(
	ctx context.Context,
	userID int64,
	role string,
	enabled bool,
	groupIDs []int64,
	localCredentialEnabled *bool,
) error {
	if role != "admin" && role != "editor" && role != "viewer" {
		return errors.New("invalid user role")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE users
SET role=$2,
    enabled=$3,
    session_version=CASE WHEN enabled AND NOT $3 THEN session_version+1 ELSE session_version END
WHERE id=$1`, userID, role, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !enabled {
		if _, err := tx.Exec(ctx, `
DELETE FROM local_sessions
WHERE user_id=$1`, userID); err != nil {
			return err
		}
	}
	if localCredentialEnabled != nil {
		if _, err := tx.Exec(ctx, `
UPDATE local_credentials
SET enabled=$2,updated_at=now()
WHERE user_id=$1`, userID, *localCredentialEnabled); err != nil {
			return err
		}
		if !*localCredentialEnabled {
			if _, err := tx.Exec(ctx, `
DELETE FROM local_sessions
WHERE user_id=$1`, userID); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM user_groups
WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if _, err := tx.Exec(ctx, `
INSERT INTO user_groups(user_id,group_id)
VALUES($1,$2)
ON CONFLICT DO NOTHING`, userID, groupID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RevokeUserSessions invalidates local and OIDC sessions for one account.
func (s *Store) RevokeUserSessions(ctx context.Context, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE users
SET session_version=session_version+1
WHERE id=$1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM local_sessions
WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Groups returns all groups and their current user counts.
func (s *Store) Groups(ctx context.Context) ([]Group, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
  g.id,
  g.name,
  count(DISTINCT ug.user_id),
  count(DISTINCT gp.id)
FROM wiki_groups g
LEFT JOIN user_groups ug ON ug.group_id=g.id
LEFT JOIN page_groups pg ON pg.group_id=g.id
LEFT JOIN pages gp ON gp.id=pg.page_id AND gp.deleted_at IS NULL
GROUP BY g.id
ORDER BY lower(g.name),g.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.Name, &group.UserCount, &group.PageCount); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// CreateGroup creates a normalized group and returns its persisted record.
func (s *Store) CreateGroup(ctx context.Context, name string) (Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Group{}, errors.New("group name is required")
	}

	var group Group
	err := s.pool.QueryRow(ctx, `
INSERT INTO wiki_groups(name)
VALUES($1)
RETURNING id,name`, name).
		Scan(&group.ID, &group.Name)
	if databaseError, ok := errors.AsType[*pgconn.PgError](err); ok && databaseError.Code == "23505" {
		return Group{}, ErrAlreadyExists
	}
	return group, err
}

// DeleteGroup removes a group and all of its user memberships.
func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM wiki_groups
WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// TagInfos returns all stored tags with page usage counts.
func (s *Store) TagInfos(ctx context.Context) ([]TagInfo, error) {
	rows, err := s.pool.Query(ctx, `
SELECT t.id,t.name,count(tp.id)
FROM tags t
LEFT JOIN page_tags pt ON pt.tag_id=t.id
LEFT JOIN pages tp ON tp.id=pt.page_id AND tp.deleted_at IS NULL
GROUP BY t.id
ORDER BY lower(t.name),t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagInfo
	for rows.Next() {
		var tag TagInfo
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.PageCount); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// DeleteTag removes a tag and its page associations.
func (s *Store) DeleteTag(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM tags
WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// User returns a user by database identifier.
func (s *Store) User(ctx context.Context, id int64) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx, `
SELECT id,username,email,display_name,role,enabled,session_version
FROM users
WHERE id=$1`, id).
		Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role, &user.Enabled, &user.SessionVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

// UserGroups returns the collaboration groups assigned to one user.
func (s *Store) UserGroups(ctx context.Context, userID int64) ([]Group, error) {
	rows, err := s.pool.Query(ctx, `
SELECT g.id,g.name
FROM wiki_groups g
JOIN user_groups ug ON ug.group_id=g.id
WHERE ug.user_id=$1
ORDER BY lower(g.name),g.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// AssignableGroups returns all groups for admins and memberships for other users.
func (s *Store) AssignableGroups(ctx context.Context, user User) ([]Group, error) {
	if user.Role == "admin" {
		return s.Groups(ctx)
	}
	return s.UserGroups(ctx, user.ID)
}

// ApplicationSettings returns the persisted application-wide settings.
func (s *Store) ApplicationSettings(ctx context.Context) (ApplicationSettings, error) {
	var settings ApplicationSettings
	err := s.pool.QueryRow(ctx, `
SELECT
  allow_user_registration,
  discussions_enabled,
  pdf_url,
  auth_mode,
  oidc_issuer,
  oidc_client_id,
  oidc_group_claim,
  oidc_group_sync,
  oidc_groups_authoritative,
	oidc_admin_group,
  trusted_username_headers,
  trusted_email_headers,
  trusted_display_name_headers,
	trusted_group_headers,
	trusted_admin_group,
  render_wiki_links,
  render_callouts,
  render_tabs,
  render_details,
  render_tables,
  render_table_styles,
  render_table_sorting,
  render_table_filtering,
  render_strikethrough,
  render_task_lists,
  render_autolinks,
  render_syntax_highlighting,
  render_content_language,
  render_coding_ligatures,
  render_mermaid,
  render_footnotes,
  render_definition_lists,
  render_typographer
FROM application_settings
WHERE singleton=true`).Scan(
		&settings.AllowUserRegistration,
		&settings.DiscussionsEnabled,
		&settings.PDFURL,
		&settings.Authentication.Mode,
		&settings.Authentication.OIDCIssuer,
		&settings.Authentication.OIDCClientID,
		&settings.Authentication.OIDCGroupClaim,
		&settings.Authentication.OIDCGroupSync,
		&settings.Authentication.OIDCGroupsAuthoritative,
		&settings.Authentication.OIDCAdminGroup,
		&settings.Authentication.TrustedUsernameHeaders,
		&settings.Authentication.TrustedEmailHeaders,
		&settings.Authentication.TrustedDisplayNameHeaders,
		&settings.Authentication.TrustedGroupHeaders,
		&settings.Authentication.TrustedAdminGroup,
		&settings.Rendering.WikiLinks,
		&settings.Rendering.Callouts,
		&settings.Rendering.Tabs,
		&settings.Rendering.Details,
		&settings.Rendering.Tables,
		&settings.Rendering.TableStyles,
		&settings.Rendering.TableSorting,
		&settings.Rendering.TableFiltering,
		&settings.Rendering.Strikethrough,
		&settings.Rendering.TaskLists,
		&settings.Rendering.Autolinks,
		&settings.Rendering.SyntaxHighlighting,
		&settings.Rendering.ContentLanguage,
		&settings.Rendering.CodingLigatures,
		&settings.Rendering.Mermaid,
		&settings.Rendering.Footnotes,
		&settings.Rendering.DefinitionLists,
		&settings.Rendering.Typographer,
	)
	return settings, err
}

// SaveApplicationSettings updates mutable application-wide settings.
func (s *Store) SaveApplicationSettings(ctx context.Context, settings ApplicationSettings) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO application_settings(singleton,allow_user_registration,discussions_enabled,updated_at)
VALUES(true,$1,$2,now())
ON CONFLICT(singleton) DO UPDATE
SET allow_user_registration=EXCLUDED.allow_user_registration,
    discussions_enabled=EXCLUDED.discussions_enabled,
    updated_at=now()`, settings.AllowUserRegistration, settings.DiscussionsEnabled)
	return err
}

// SavePDFSettings updates the persisted HTML-to-PDF rendering endpoint.
func (s *Store) SavePDFSettings(ctx context.Context, pdfURL string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE application_settings
SET pdf_url=$1,
    updated_at=now()
WHERE singleton=true`, pdfURL)
	return err
}

// SaveAuthenticationSettings updates non-secret browser authentication settings.
func (s *Store) SaveAuthenticationSettings(ctx context.Context, settings AuthenticationSettings) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// A changed assertion source invalidates previously observed external role state.
	if _, err := tx.Exec(ctx, `
UPDATE users
SET oidc_admin_observed=false,oidc_external_admin=false
WHERE EXISTS (
  SELECT 1
  FROM application_settings
  WHERE singleton=true
    AND (oidc_admin_group IS DISTINCT FROM $1 OR oidc_group_claim IS DISTINCT FROM $2)
)`, settings.OIDCAdminGroup, settings.OIDCGroupClaim); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE users
SET trusted_proxy_admin_observed=false,trusted_proxy_external_admin=false
WHERE EXISTS (
  SELECT 1
  FROM application_settings
  WHERE singleton=true
    AND (trusted_admin_group IS DISTINCT FROM $1 OR trusted_group_headers IS DISTINCT FROM $2)
)`, settings.TrustedAdminGroup, settings.TrustedGroupHeaders); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
UPDATE application_settings
SET auth_mode=$1,
    oidc_issuer=$2,
    oidc_client_id=$3,
    oidc_group_claim=$4,
    oidc_group_sync=$5,
    oidc_groups_authoritative=$6,
	oidc_admin_group=$7,
    trusted_username_headers=$8,
    trusted_email_headers=$9,
    trusted_display_name_headers=$10,
	trusted_group_headers=$11,
	trusted_admin_group=$12,
    updated_at=now()
WHERE singleton=true`,
		settings.Mode,
		settings.OIDCIssuer,
		settings.OIDCClientID,
		settings.OIDCGroupClaim,
		settings.OIDCGroupSync,
		settings.OIDCGroupsAuthoritative,
		settings.OIDCAdminGroup,
		settings.TrustedUsernameHeaders,
		settings.TrustedEmailHeaders,
		settings.TrustedDisplayNameHeaders,
		settings.TrustedGroupHeaders,
		settings.TrustedAdminGroup,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
DELETE FROM oidc_group_mappings`); err != nil {
		return err
	}
	for _, mapping := range settings.OIDCGroupMappings {
		if _, err := tx.Exec(ctx, `
INSERT INTO oidc_group_mappings(oidc_group,group_id)
VALUES($1,$2)`, strings.TrimSpace(mapping.OIDCGroup), mapping.GroupID); err != nil {
			if databaseError, ok := errors.AsType[*pgconn.PgError](err); ok && databaseError.Code == "23503" {
				return ErrNotFound
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

// SaveRenderingSettings updates administrator-controlled Markdown rendering features.
func (s *Store) SaveRenderingSettings(ctx context.Context, settings RenderingSettings) error {
	_, err := s.pool.Exec(ctx, `
UPDATE application_settings
SET render_wiki_links=$1,
    render_callouts=$2,
    render_tabs=$3,
    render_details=$4,
    render_tables=$5,
    render_table_styles=$6,
    render_table_sorting=$7,
    render_table_filtering=$8,
    render_strikethrough=$9,
    render_task_lists=$10,
    render_autolinks=$11,
    render_syntax_highlighting=$12,
    render_content_language=$13,
    render_coding_ligatures=$14,
    render_mermaid=$15,
    render_footnotes=$16,
    render_definition_lists=$17,
    render_typographer=$18,
    updated_at=now()
WHERE singleton=true`,
		settings.WikiLinks,
		settings.Callouts,
		settings.Tabs,
		settings.Details,
		settings.Tables,
		settings.TableStyles,
		settings.TableSorting,
		settings.TableFiltering,
		settings.Strikethrough,
		settings.TaskLists,
		settings.Autolinks,
		settings.SyntaxHighlighting,
		settings.ContentLanguage,
		settings.CodingLigatures,
		settings.Mermaid,
		settings.Footnotes,
		settings.DefinitionLists,
		settings.Typographer,
	)
	return err
}

// SearchUsers returns accounts matching a username, display name, or email prefix/substring.
func (s *Store) SearchUsers(ctx context.Context, query string, limit int) ([]User, error) {
	query = strings.TrimSpace(query)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id,username,email,display_name,role
FROM users
WHERE username ILIKE $1 OR display_name ILIKE $1 OR email ILIKE $1
ORDER BY
  CASE WHEN username ILIKE $2 OR display_name ILIKE $2 THEN 0 ELSE 1 END,
  lower(display_name),lower(username),id
LIMIT $3`, "%"+query+"%", query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// GroupMembers returns accounts assigned to a group.
func (s *Store) GroupMembers(ctx context.Context, groupID int64) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
SELECT u.id,u.username,u.email,u.display_name,u.role
FROM users u
JOIN user_groups ug ON ug.user_id=u.id
WHERE ug.group_id=$1
ORDER BY lower(u.display_name),lower(u.username),u.id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// AddGroupMember assigns a user to a group.
func (s *Store) AddGroupMember(ctx context.Context, groupID, userID int64) error {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO user_groups(user_id,group_id)
SELECT u.id,g.id FROM users u CROSS JOIN wiki_groups g
WHERE u.id=$2 AND g.id=$1
ON CONFLICT DO NOTHING`, groupID, userID)
	if err == nil && tag.RowsAffected() == 0 {
		var exists bool
		err = s.pool.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM user_groups WHERE user_id=$2 AND group_id=$1)`, groupID, userID).
			Scan(&exists)
		if err == nil && !exists {
			return ErrNotFound
		}
	}
	return err
}

// RemoveGroupMember removes a user from a group.
func (s *Store) RemoveGroupMember(ctx context.Context, groupID, userID int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM user_groups
WHERE group_id=$1 AND user_id=$2`, groupID, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

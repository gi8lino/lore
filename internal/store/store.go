package store

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/internal/revision"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Store provides PostgreSQL-backed persistence for wiki data.
type Store struct {
	// pool is the PostgreSQL connection pool used by store operations.
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL and applies pending embedded migrations.
func Open(ctx context.Context, url string, logger *slog.Logger) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	s := &Store{pool: pool}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := s.migrate(ctx, logger); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the PostgreSQL connection pool.
func (s *Store) Close() { s.pool.Close() }

// Ping verifies that PostgreSQL is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// migrate applies unapplied embedded SQL migrations in version order.
func (s *Store) migrate(ctx context.Context, logger *slog.Logger) error {
	// Keep migration history in the same database so startup can safely skip
	// schema changes that have already been committed.
	if _, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}

	// Sort embedded entries by their numeric filename prefix because lexical
	// filename order would place migration 10 before migration 2.
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		left, _ := strconv.Atoi(strings.SplitN(entries[i].Name(), "_", 2)[0])
		right, _ := strconv.Atoi(strings.SplitN(entries[j].Name(), "_", 2)[0])
		return left < right
	})

	for _, e := range entries {
		if !isSQLFile(e) {
			continue
		}

		// Extract the migration version number from the filename.
		v, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("invalid migration %s", e.Name())
		}

		var exists bool
		if err := s.pool.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, v).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		sql, err := migrationFiles.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}

		// Apply the schema change and record its version atomically. A failed
		// statement therefore remains eligible for retry on the next startup.
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err == nil {
			_, err = tx.Exec(ctx, `
INSERT INTO schema_migrations(version)
VALUES($1)`, v)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %d: %w", v, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		logger.Info(
			"applied database migration",
			"event", "database_migration_applied",
			"version", v,
			"migration", e.Name(),
		)
	}
	return nil
}

// isSQLFile reports whether e is a regular SQL migration file.
func isSQLFile(e fs.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".sql")
}

// EnsureAdministrator creates or refreshes the administrator used by no-auth mode.
func (s *Store) EnsureAdministrator(ctx context.Context, username, email, displayName string) (User, error) {
	if displayName == "" {
		displayName = username
	}

	var user User
	err := s.pool.QueryRow(ctx, `
INSERT INTO users(username,email,display_name,role,last_login)
VALUES($1,$2,$3,'admin',now())
ON CONFLICT(username) DO UPDATE SET
  email=EXCLUDED.email,
  display_name=EXCLUDED.display_name,
  role='admin',
  last_login=now()
RETURNING id,username,email,display_name,role`, username, email, displayName).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.Role,
	)
	return user, err
}

// TrustedProxyUser refreshes a trusted-proxy user by username or creates one when registration is enabled.
func (s *Store) TrustedProxyUser(ctx context.Context, username, email, displayName string) (User, error) {
	if displayName == "" {
		displayName = username
	}

	var user User
	err := s.pool.QueryRow(ctx, `
UPDATE users
SET email=$2,display_name=$3,last_login=now()
WHERE username=$1
RETURNING id,username,email,display_name,role`, username, email, displayName).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	settings, err := s.ApplicationSettings(ctx)
	if err != nil {
		return User{}, err
	}
	if !settings.AllowUserRegistration {
		return User{}, ErrRegistrationDisabled
	}

	err = s.pool.QueryRow(ctx, `
INSERT INTO users(username,email,display_name,last_login)
VALUES($1,$2,$3,now())
ON CONFLICT(username) DO UPDATE
SET email=EXCLUDED.email,display_name=EXCLUDED.display_name,last_login=now()
RETURNING id,username,email,display_name,role`, username, email, displayName).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Role)
	return user, err
}

// UserByToken authenticates an API bearer token and updates its last-used timestamp.
func (s *Store) UserByToken(ctx context.Context, token string) (User, error) {
	h := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(h[:])
	var u User
	err := s.pool.QueryRow(ctx, `
UPDATE api_tokens t
SET last_used=now() FROM users u
WHERE t.token_hash=$1 AND coalesce(t.user_id,t.created_by)=u.id AND (t.expires_at IS NULL OR t.expires_at>now())
RETURNING u.id,u.username,u.email,u.display_name,u.role`, hash).
		Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return u, err
}

const pageSelect = `
SELECT p.id,p.slug,p.title,coalesce(max(ni.icon),''),p.markdown_content,coalesce(p.created_by,0),coalesce(p.updated_by,0),coalesce(u.display_name,u.username,''),p.created_at,p.updated_at,p.view_count,coalesce(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),'{}')
FROM pages p
LEFT JOIN navigation_icons ni ON ni.path=p.slug
LEFT JOIN users u ON u.id=p.updated_by
LEFT JOIN page_tags pt ON pt.page_id=p.id
LEFT JOIN tags t ON t.id=pt.tag_id`

// scanPage scans the common page projection and normalizes missing rows.
func scanPage(row pgx.Row) (Page, error) {
	var p Page
	err := row.Scan(
		&p.ID,
		&p.Slug,
		&p.Title,
		&p.Icon,
		&p.Markdown,
		&p.CreatedBy,
		&p.UpdatedBy,
		&p.Author,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ViewCount,
		&p.Tags,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return p, err
}

// GetPage returns a wiki page by slug.
func (s *Store) GetPage(ctx context.Context, slug string) (Page, error) {
	page, err := scanPage(
		s.pool.QueryRow(ctx, pageSelect+`
WHERE p.slug=$1 AND p.deleted_at IS NULL
GROUP BY p.id,u.id`, slug),
	)
	if err != nil {
		return Page{}, err
	}
	page.Groups, err = s.PageGroups(ctx, page.ID)
	if err != nil {
		return Page{}, err
	}
	if err := s.pool.QueryRow(ctx, `
SELECT p.content_language,p.status,coalesce(p.owner_group_id,0),coalesce(g.name,''),p.last_reviewed_at,p.review_interval_days,p.deprecated_target
FROM pages p
LEFT JOIN wiki_groups g ON g.id=p.owner_group_id
WHERE p.id=$1`, page.ID).Scan(
		&page.Language,
		&page.Status,
		&page.OwnerGroupID,
		&page.OwnerGroup,
		&page.LastReviewedAt,
		&page.ReviewIntervalDays,
		&page.DeprecatedTarget,
	); err != nil {
		return Page{}, err
	}
	page.Properties, err = s.PageProperties(ctx, page.ID)
	if err != nil {
		return Page{}, err
	}
	return page, nil
}

// ListPages returns recently updated wiki pages up to the requested limit.
func (s *Store) ListPages(ctx context.Context, limit int) ([]Page, error) {
	rows, err := s.pool.Query(
		ctx,
		pageSelect+`
WHERE p.deleted_at IS NULL
GROUP BY p.id,u.id
ORDER BY p.updated_at DESC
LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPages(rows)
}

// NavigationPages returns the minimal page data required to build navigation.
func (s *Store) NavigationPages(ctx context.Context) ([]Page, error) {
	rows, err := s.pool.Query(ctx, `
SELECT p.slug,p.title,coalesce(i.icon,'')
FROM pages p
LEFT JOIN navigation_icons i ON i.path=p.slug
WHERE p.deleted_at IS NULL
ORDER BY p.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		var page Page
		if err := rows.Scan(&page.Slug, &page.Title, &page.Icon); err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, rows.Err()
}

// collectPages scans all rows from a common page query.
func collectPages(rows pgx.Rows) ([]Page, error) {
	var out []Page
	for rows.Next() {
		p, e := scanPage(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SavePage persists page content, revision history, tags, and links transactionally.
func (s *Store) SavePage(
	ctx context.Context,
	previousSlug, slug, title, icon, language, markdown, message string,
	tags, links []string,
	groupIDs []int64,
	metadata PageMetadata,
	properties map[string]string,
	user User,
) (Page, error) {
	if !ValidPageStatus(metadata.Status) {
		return Page{}, errors.New("invalid page status")
	}
	if metadata.ReviewIntervalDays < 0 {
		return Page{}, errors.New("invalid review interval")
	}
	metadata.DeprecatedTarget = strings.TrimSpace(metadata.DeprecatedTarget)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := validateAssignableGroup(ctx, tx, metadata.OwnerGroupID, user); err != nil {
		return Page{}, err
	}

	lookupSlug := slug
	if strings.TrimSpace(previousSlug) != "" {
		lookupSlug = strings.TrimSpace(previousSlug)
	}

	var id int64
	var deleted bool
	err = tx.QueryRow(ctx, `
SELECT id,deleted_at IS NOT NULL
FROM pages
WHERE slug=$1 FOR UPDATE`, lookupSlug).
		Scan(&id, &deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		var aliasExists bool
		if aliasErr := tx.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM page_aliases WHERE alias=$1)`, slug).Scan(&aliasExists); aliasErr != nil {
			return Page{}, aliasErr
		}
		if aliasExists {
			return Page{}, ErrAlreadyExists
		}
		err = tx.QueryRow(ctx, `
INSERT INTO pages(
  slug,title,content_language,markdown_content,created_by,updated_by,status,owner_group_id,last_reviewed_at,review_interval_days,deprecated_target
) VALUES(
  $1,$2,$3,$4,$5,$5,$6,NULLIF($7,0),CASE WHEN $8 THEN now() ELSE NULL END,$9,$10
) RETURNING id`,
			slug, title, language, markdown, user.ID, metadata.Status, metadata.OwnerGroupID, metadata.MarkReviewed, metadata.ReviewIntervalDays, metadata.DeprecatedTarget,
		).Scan(&id)
	} else if err == nil && deleted {
		return Page{}, ErrPageInBin
	} else if err == nil {
		if lookupSlug != slug {
			var conflict bool
			if err = tx.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM pages WHERE slug=$1 AND id<>$2) OR EXISTS(SELECT 1 FROM page_aliases WHERE alias=$1 AND page_id<>$2)`, slug, id).Scan(&conflict); err != nil {
				return Page{}, err
			}
			if conflict {
				return Page{}, ErrAlreadyExists
			}
			if _, err = tx.Exec(ctx, `
UPDATE pages
SET slug=$2
WHERE id=$1`, id, slug); err != nil {
				return Page{}, err
			}
			if _, err = tx.Exec(ctx, `
UPDATE navigation_icons
SET path=$2
WHERE path=$1`, lookupSlug, slug); err != nil {
				return Page{}, err
			}
			if _, err = tx.Exec(ctx, `
INSERT INTO page_aliases(alias,page_id)
VALUES($1,$2)
ON CONFLICT(alias) DO UPDATE
SET page_id=EXCLUDED.page_id`, lookupSlug, id); err != nil {
				return Page{}, err
			}
		}
		_, err = tx.Exec(ctx, `
UPDATE pages
SET title=$2,content_language=$3,markdown_content=$4,updated_by=$5,updated_at=now(),
    status=$6,owner_group_id=NULLIF($7,0),
    last_reviewed_at=CASE WHEN $8 THEN now() ELSE last_reviewed_at END,
    review_interval_days=$9,deprecated_target=$10
WHERE id=$1`,
			id, title, language, markdown, user.ID, metadata.Status, metadata.OwnerGroupID, metadata.MarkReviewed, metadata.ReviewIntervalDays, metadata.DeprecatedTarget,
		)
	}
	if err != nil {
		return Page{}, err
	}

	icon = strings.TrimSpace(icon)
	if icon == "" {
		if _, err = tx.Exec(ctx, `
DELETE FROM navigation_icons
WHERE path=$1`, slug); err != nil {
			return Page{}, err
		}
	} else if _, err = tx.Exec(ctx, `
INSERT INTO navigation_icons(path,icon)
VALUES($1,$2)
ON CONFLICT(path) DO UPDATE SET icon=EXCLUDED.icon`, slug, icon); err != nil {
		return Page{}, err
	}

	var rev int
	if err = tx.QueryRow(ctx, `
SELECT coalesce(max(revision_number),0)+1
FROM page_revisions
WHERE page_id=$1`, id).Scan(&rev); err != nil {
		return Page{}, err
	}
	if _, err = tx.Exec(ctx, `
INSERT INTO page_revisions(page_id,revision_number,markdown_content,created_by,message)
VALUES($1,$2,$3,$4,$5)`, id, rev, markdown, user.ID, message); err != nil {
		return Page{}, err
	}

	if _, err = tx.Exec(ctx, `
DELETE FROM page_tags
WHERE page_id=$1`, id); err != nil {
		return Page{}, err
	}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		var tagID int64
		if err = tx.QueryRow(ctx, `
INSERT INTO tags(name)
VALUES($1)
ON CONFLICT(name) DO UPDATE
SET name=EXCLUDED.name
RETURNING id`, tag).Scan(&tagID); err != nil {
			return Page{}, err
		}
		if _, err = tx.Exec(ctx, `
INSERT INTO page_tags(page_id,tag_id)
VALUES($1,$2)
ON CONFLICT DO NOTHING`, id, tagID); err != nil {
			return Page{}, err
		}
	}

	if err = replacePageGroups(ctx, tx, id, groupIDs, user); err != nil {
		return Page{}, err
	}

	if err = replacePageProperties(ctx, tx, id, properties); err != nil {
		return Page{}, err
	}

	if _, err = tx.Exec(ctx, `
DELETE FROM page_links
WHERE source_page_id=$1`, id); err != nil {
		return Page{}, err
	}
	for _, link := range links {
		if _, err = tx.Exec(ctx, `
INSERT INTO page_links(source_page_id,target_slug)
VALUES($1,$2)
ON CONFLICT DO NOTHING`, id, link); err != nil {
			return Page{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return Page{}, err
	}
	return s.GetPage(ctx, slug)
}

// DeletePage moves a wiki page into the recycle bin.
func (s *Store) DeletePage(ctx context.Context, slug string, userID int64) error {
	tag, err := s.pool.Exec(
		ctx,
		`
UPDATE pages
SET deleted_at=now(),deleted_by=$2
WHERE slug=$1 AND deleted_at IS NULL`,
		slug,
		userID,
	)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeletedPages returns pages currently held in the recycle bin, newest deletion first.
func (s *Store) DeletedPages(ctx context.Context) ([]DeletedPage, error) {
	rows, err := s.pool.Query(ctx, `
SELECT p.id,p.slug,p.title,coalesce(max(ni.icon),''),p.markdown_content,coalesce(p.created_by,0),coalesce(p.updated_by,0),
       coalesce(editor.display_name,editor.username,''),p.created_at,p.updated_at,p.view_count,
       coalesce(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),'{}'),
       p.deleted_at,coalesce(deleter.display_name,deleter.username,'')
FROM pages p
LEFT JOIN navigation_icons ni ON ni.path=p.slug
LEFT JOIN users editor ON editor.id=p.updated_by
LEFT JOIN users deleter ON deleter.id=p.deleted_by
LEFT JOIN page_tags pt ON pt.page_id=p.id
LEFT JOIN tags t ON t.id=pt.tag_id
WHERE p.deleted_at IS NOT NULL
GROUP BY p.id,editor.id,deleter.id
ORDER BY p.deleted_at DESC,p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []DeletedPage
	for rows.Next() {
		var item DeletedPage
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Icon, &item.Markdown, &item.CreatedBy, &item.UpdatedBy, &item.Author, &item.CreatedAt, &item.UpdatedAt, &item.ViewCount, &item.Tags, &item.DeletedAt, &item.DeletedBy); err != nil {
			return nil, err
		}
		pages = append(pages, item)
	}
	return pages, rows.Err()
}

// RestorePage restores a page from the recycle bin.
func (s *Store) RestorePage(ctx context.Context, slug string) error {
	tag, err := s.pool.Exec(
		ctx,
		`
UPDATE pages
SET deleted_at=NULL,deleted_by=NULL
WHERE slug=$1 AND deleted_at IS NOT NULL`,
		slug,
	)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// PermanentlyDeletePage removes one page already held in the recycle bin.
func (s *Store) PermanentlyDeletePage(ctx context.Context, slug string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
DELETE FROM pages
WHERE slug=$1
  AND deleted_at IS NOT NULL`, slug)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
DELETE FROM navigation_icons AS icon
WHERE NOT EXISTS (
  SELECT 1
  FROM pages
  WHERE slug=icon.path
     OR slug LIKE icon.path || '/%'
)`); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RecordView increments page views and records the user's most recent view.
func (s *Store) RecordView(ctx context.Context, slug string, userID int64) error {
	_, err := s.pool.Exec(
		ctx,
		`
WITH p AS (UPDATE pages SET view_count=view_count+1 WHERE slug=$1 AND deleted_at IS NULL RETURNING id) INSERT INTO page_views(user_id,page_id,viewed_at) SELECT $2,id,now()
FROM p
ON CONFLICT(user_id,page_id) DO UPDATE
SET viewed_at=now()`,
		slug,
		userID,
	)
	return err
}

// SetFavorite adds or removes a page from a user's favorites.
func (s *Store) SetFavorite(ctx context.Context, slug string, userID int64, on bool) error {
	if on {
		_, err := s.pool.Exec(
			ctx,
			`
INSERT INTO favorites(user_id,page_id)
SELECT $2,id FROM pages
WHERE slug=$1 AND deleted_at IS NULL
ON CONFLICT DO NOTHING`,
			slug,
			userID,
		)
		return err
	}
	_, err := s.pool.Exec(
		ctx,
		`
DELETE FROM favorites f
USING pages p
WHERE f.page_id=p.id AND p.slug=$1 AND p.deleted_at IS NULL AND f.user_id=$2`,
		slug,
		userID,
	)
	return err
}

// IsFavorite reports whether a page is currently pinned as a favorite by the user.
func (s *Store) IsFavorite(ctx context.Context, slug string, userID int64) (bool, error) {
	var favorite bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM favorites f
  JOIN pages p ON p.id=f.page_id
  WHERE f.user_id=$2 AND p.slug=$1 AND p.deleted_at IS NULL
)`, slug, userID).Scan(&favorite)
	return favorite, err
}

// Favorites returns a user's favorite pages in newest-first order.
func (s *Store) Favorites(ctx context.Context, userID int64) ([]Page, error) {
	rows, err := s.pool.Query(
		ctx,
		pageSelect+`
JOIN favorites f ON f.page_id=p.id AND f.user_id=$1
WHERE p.deleted_at IS NULL
GROUP BY p.id,u.id,f.created_at
ORDER BY f.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPages(rows)
}

// Backlinks returns pages that reference the supplied page slug.
func (s *Store) Backlinks(ctx context.Context, slug string) ([]Page, error) {
	base := slug
	if index := strings.LastIndex(slug, "/"); index >= 0 {
		base = slug[index+1:]
	}
	rows, err := s.pool.Query(
		ctx,
		pageSelect+`
JOIN page_links l ON l.source_page_id=p.id AND l.target_slug IN ($1,$2)
WHERE p.deleted_at IS NULL
GROUP BY p.id,u.id
ORDER BY p.title`,
		slug,
		base,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPages(rows)
}

// ResolvePageAlias resolves a historical page path to its current active slug.
func (s *Store) ResolvePageAlias(ctx context.Context, alias string) (string, error) {
	var slug string
	err := s.pool.QueryRow(ctx, `
SELECT p.slug
FROM page_aliases a
JOIN pages p ON p.id=a.page_id AND p.deleted_at IS NULL
WHERE a.alias=$1`, alias).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return slug, err
}

// LatestRevision returns the newest revision and total count, or a zero count when none exist.
func (s *Store) LatestRevision(ctx context.Context, slug string) (revision.Revision, int, error) {
	const query = `
SELECT
  r.id,
  r.revision_number,
  coalesce(u.display_name,u.username,''),
  r.created_at,
  r.message,
  r.markdown_content,
  coalesce((SELECT previous.markdown_content FROM page_revisions previous WHERE previous.page_id=r.page_id AND previous.revision_number=r.revision_number-1),''),
  count(*) OVER()
FROM page_revisions r
JOIN pages p ON p.id=r.page_id
LEFT JOIN users u ON u.id=r.created_by
WHERE p.slug=$1 AND p.deleted_at IS NULL
ORDER BY r.revision_number DESC
LIMIT 1`

	var record revision.Revision
	var markdown, previous string
	var count int
	err := s.pool.QueryRow(ctx, query, slug).Scan(
		&record.ID,
		&record.Number,
		&record.Author,
		&record.CreatedAt,
		&record.Message,
		&markdown,
		&previous,
		&count,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return revision.Revision{}, 0, nil
	}
	if err == nil {
		record.Markdown = markdown
		record.PreviousMarkdown = previous
	}
	return record, count, err
}

// Revision returns one persisted page revision by revision number.
func (s *Store) Revision(ctx context.Context, slug string, number int) (revision.Revision, error) {
	var record revision.Revision
	err := s.pool.QueryRow(ctx, `
SELECT r.id,r.revision_number,coalesce(u.display_name,u.username,''),r.created_at,r.message,r.markdown_content
FROM page_revisions r
JOIN pages p ON p.id=r.page_id
LEFT JOIN users u ON u.id=r.created_by
WHERE p.slug=$1 AND p.deleted_at IS NULL AND r.revision_number=$2`, slug, number).Scan(
		&record.ID,
		&record.Number,
		&record.Author,
		&record.CreatedAt,
		&record.Message,
		&record.Markdown,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return revision.Revision{}, ErrNotFound
	}
	return record, err
}

// Revisions returns a page's revision metadata in newest-first order.
func (s *Store) Revisions(ctx context.Context, slug string) ([]revision.Revision, error) {
	const query = `
WITH history AS (
  SELECT
    r.id,
    r.revision_number,
    r.created_by,
    r.created_at,
    r.message,
    r.markdown_content,
    lag(r.markdown_content,1,'') OVER (ORDER BY r.revision_number) AS previous_markdown
  FROM page_revisions r
  JOIN pages p ON p.id=r.page_id
  WHERE p.slug=$1 AND p.deleted_at IS NULL
)
SELECT h.id,h.revision_number,coalesce(u.display_name,u.username,''),h.created_at,h.message,h.markdown_content,h.previous_markdown
FROM history h
LEFT JOIN users u ON u.id=h.created_by
ORDER BY h.revision_number DESC`

	rows, err := s.pool.Query(ctx, query, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []revision.Revision
	for rows.Next() {
		var record revision.Revision
		if err := rows.Scan(
			&record.ID,
			&record.Number,
			&record.Author,
			&record.CreatedAt,
			&record.Message,
			&record.Markdown,
			&record.PreviousMarkdown,
		); err != nil {
			return nil, err
		}
		revisions = append(revisions, record)
	}
	return revisions, rows.Err()
}

// Tags returns all known tags as JSON.
func (s *Store) Tags(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT name
FROM tags
ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

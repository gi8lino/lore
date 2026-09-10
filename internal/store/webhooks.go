package store

import (
	"context"
	"errors"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/jackc/pgx/v5"
)

const webhookSelect = `
SELECT id,name,url,events,secret,enabled,created_at,updated_at
FROM webhooks`

type webhookRow interface {
	Scan(...any) error
}

// Webhooks returns all outgoing webhook configurations in name order.
func (s *Store) Webhooks(ctx context.Context) ([]domain.Webhook, error) {
	rows, err := s.pool.Query(ctx, webhookSelect+`
ORDER BY lower(name),id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Webhook

	for rows.Next() {
		item, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, rows.Err()
}

// Webhook returns one outgoing webhook by identifier.
func (s *Store) Webhook(ctx context.Context, id int64) (domain.Webhook, error) {
	item, err := scanWebhook(s.pool.QueryRow(ctx, webhookSelect+`
WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Webhook{}, domain.ErrNotFound
	}

	return item, err
}

// SaveWebhook creates or replaces one outgoing webhook.
func (s *Store) SaveWebhook(ctx context.Context, id int64, item domain.Webhook) (domain.Webhook, error) {
	if id == 0 {
		return s.createWebhook(ctx, item)
	}

	return s.updateWebhook(ctx, id, item)
}

// createWebhook inserts one outgoing webhook and returns the persisted row.
func (s *Store) createWebhook(ctx context.Context, item domain.Webhook) (domain.Webhook, error) {
	row := s.pool.QueryRow(ctx, `
INSERT INTO webhooks(name,url,events,secret,enabled)
VALUES($1,$2,$3,$4,$5)
RETURNING id,name,url,events,secret,enabled,created_at,updated_at`,
		item.Name,
		item.URL,
		item.Events,
		item.Secret,
		item.Enabled,
	)

	saved, err := scanWebhook(row)

	return saved, mutationError(err)
}

// updateWebhook replaces one outgoing webhook and returns the persisted row.
func (s *Store) updateWebhook(ctx context.Context, id int64, item domain.Webhook) (domain.Webhook, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE webhooks
SET name=$2,url=$3,events=$4,secret=$5,enabled=$6,updated_at=now()
WHERE id=$1
RETURNING id,name,url,events,secret,enabled,created_at,updated_at`,
		id,
		item.Name,
		item.URL,
		item.Events,
		item.Secret,
		item.Enabled,
	)

	saved, err := scanWebhook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Webhook{}, domain.ErrNotFound
	}

	return saved, mutationError(err)
}

// scanWebhook decodes one outgoing webhook row.
func scanWebhook(row webhookRow) (domain.Webhook, error) {
	var item domain.Webhook

	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.URL,
		&item.Events,
		&item.Secret,
		&item.Enabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	return item, err
}

// DeleteWebhook removes one outgoing webhook.
func (s *Store) DeleteWebhook(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM webhooks WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return err
}

// AddWebhookDelivery records the outcome of one outgoing delivery attempt.
func (s *Store) AddWebhookDelivery(ctx context.Context, webhookID int64, event string, statusCode int, message string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO webhook_deliveries(webhook_id,event,status_code,error)
VALUES($1,$2,$3,$4)`, webhookID, event, statusCode, message)

	return err
}

// WebhookDeliveries returns the newest outgoing delivery attempts.
func (s *Store) WebhookDeliveries(ctx context.Context, limit int) ([]domain.WebhookDelivery, error) {
	rows, err := s.pool.Query(ctx, `
SELECT d.id,d.webhook_id,w.name,d.event,d.status_code,d.error,d.created_at
FROM webhook_deliveries d
JOIN webhooks w ON w.id=d.webhook_id
ORDER BY d.created_at DESC,d.id DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.WebhookDelivery

	for rows.Next() {
		var item domain.WebhookDelivery

		if err := rows.Scan(
			&item.ID,
			&item.WebhookID,
			&item.WebhookName,
			&item.Event,
			&item.StatusCode,
			&item.Error,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, rows.Err()
}

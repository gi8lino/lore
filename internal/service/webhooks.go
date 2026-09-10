package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/secrets"
)

var webhookEvents = []string{
	"page.created", "page.updated", "page.renamed", "page.deleted", "page.moved",
	"page.reviewed", "page.review_requested", "page.review_approved", "page.review_changes_requested",
	"page.revision_restored", "comment.created", "pages.imported",
	"page.bulk_status", "page.bulk_tag", "page.bulk_group", "page.bulk_move", "page.bulk_delete",
}

// OutgoingEvent is the stable payload emitted by application mutations.
type OutgoingEvent struct {
	Event      string    `json:"event"`
	ActorID    int64     `json:"actor_id,omitempty"`
	ObjectType string    `json:"object_type"`
	ObjectKey  string    `json:"object_key"`
	Detail     string    `json:"detail,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// EventSink receives committed application events as best-effort side effects.
type EventSink interface {
	Emit(context.Context, OutgoingEvent) error
}

type webhookRepository interface {
	Webhooks(context.Context) ([]domain.Webhook, error)
	Webhook(context.Context, int64) (domain.Webhook, error)
	SaveWebhook(context.Context, int64, domain.Webhook) (domain.Webhook, error)
	DeleteWebhook(context.Context, int64) error
	AddWebhookDelivery(context.Context, int64, string, int, string) error
	WebhookDeliveries(context.Context, int) ([]domain.WebhookDelivery, error)
}

type WebhookInput struct {
	Name        string
	URL         string
	Events      []string
	Secret      string
	ClearSecret bool
	Enabled     bool
}

// Webhooks owns encrypted configuration and signed HTTP delivery.
type Webhooks struct {
	repository webhookRepository
	secrets    *secrets.Cipher
	client     *http.Client
	logger     *slog.Logger
}

func NewWebhooks(repository webhookRepository, secretCipher *secrets.Cipher, logger *slog.Logger) *Webhooks {
	return &Webhooks{repository: repository, secrets: secretCipher, client: &http.Client{Timeout: 5 * time.Second}, logger: logger}
}

func WebhookEvents() []string { return slices.Clone(webhookEvents) }

func (s *Webhooks) Webhooks(ctx context.Context) ([]domain.Webhook, error) {
	items, err := s.repository.Webhooks(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].SecretConfigured = items[index].Secret != ""
		items[index].Secret = ""
	}
	return items, nil
}

func (s *Webhooks) WebhookDeliveries(ctx context.Context, limit int) ([]domain.WebhookDelivery, error) {
	return s.repository.WebhookDeliveries(ctx, limit)
}

func (s *Webhooks) SaveWebhook(ctx context.Context, id int64, input WebhookInput) (domain.Webhook, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	validation := &ValidationError{}
	if input.Name == "" {
		validation.Fields = append(validation.Fields, FieldError{Field: "name", Message: "A webhook name is required."})
	}
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		validation.Fields = append(validation.Fields, FieldError{Field: "url", Message: "Enter an absolute HTTP or HTTPS URL."})
	}
	events := normalizeWebhookEvents(input.Events)
	if len(events) == 0 {
		validation.Fields = append(validation.Fields, FieldError{Field: "events", Message: "Choose at least one event."})
	}
	if len(validation.Fields) > 0 {
		return domain.Webhook{}, validation
	}

	secretValue := ""
	if id != 0 && !input.ClearSecret {
		existing, err := s.repository.Webhook(ctx, id)
		if err != nil {
			return domain.Webhook{}, err
		}
		secretValue = existing.Secret
	}
	if input.ClearSecret {
		secretValue = ""
	}
	if strings.TrimSpace(input.Secret) != "" {
		if s.secrets == nil || !s.secrets.Configured() {
			return domain.Webhook{}, newValidationError("secret", "Configure LORE__ENCRYPTION_KEY before saving a signing secret.")
		}
		secretValue, err = s.secrets.Encrypt(input.Secret)
		if err != nil {
			return domain.Webhook{}, err
		}
	}
	item, err := s.repository.SaveWebhook(ctx, id, domain.Webhook{Name: input.Name, URL: input.URL, Events: events, Secret: secretValue, Enabled: input.Enabled})
	if err != nil {
		return domain.Webhook{}, err
	}
	item.SecretConfigured = item.Secret != ""
	item.Secret = ""
	return item, nil
}

func (s *Webhooks) DeleteWebhook(ctx context.Context, id int64) error {
	return s.repository.DeleteWebhook(ctx, id)
}

func (s *Webhooks) TestWebhook(ctx context.Context, id int64) error {
	item, err := s.repository.Webhook(ctx, id)
	if err != nil {
		return err
	}
	return s.deliver(ctx, item, OutgoingEvent{Event: "webhook.test", ObjectType: "webhook", ObjectKey: item.Name, Detail: "Test delivery from Lore", OccurredAt: time.Now().UTC()})
}

func (s *Webhooks) Emit(ctx context.Context, event OutgoingEvent) error {
	items, err := s.repository.Webhooks(ctx)
	if err != nil {
		return err
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	var combined error
	for _, item := range items {
		if !item.Enabled || !slices.Contains(item.Events, event.Event) {
			continue
		}
		if err := s.deliver(ctx, item, event); err != nil {
			combined = errors.Join(combined, err)
		}
	}
	return combined
}

func (s *Webhooks) deliver(ctx context.Context, item domain.Webhook, event OutgoingEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, item.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lore-Webhook/1")
	req.Header.Set("X-Lore-Event", event.Event)
	if item.Secret != "" {
		if s.secrets == nil || !s.secrets.Configured() {
			err = secrets.ErrNotConfigured
			_ = s.repository.AddWebhookDelivery(ctx, item.ID, event.Event, 0, err.Error())
			return err
		}
		plain, decryptErr := s.secrets.Decrypt(item.Secret)
		if decryptErr != nil {
			_ = s.repository.AddWebhookDelivery(ctx, item.ID, event.Event, 0, decryptErr.Error())
			return decryptErr
		}
		mac := hmac.New(sha256.New, []byte(plain))
		_, _ = mac.Write(body)
		req.Header.Set("X-Lore-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	response, err := s.client.Do(req)
	if err != nil {
		_ = s.repository.AddWebhookDelivery(ctx, item.ID, event.Event, 0, err.Error())
		return fmt.Errorf("deliver webhook %q: %w", item.Name, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	message := ""
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message = response.Status
		err = fmt.Errorf("webhook %q returned %s", item.Name, response.Status)
	}
	if recordErr := s.repository.AddWebhookDelivery(ctx, item.ID, event.Event, response.StatusCode, message); recordErr != nil && err == nil {
		err = recordErr
	}
	return err
}

func normalizeWebhookEvents(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if slices.Contains(webhookEvents, value) && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

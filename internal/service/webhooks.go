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

// WebhookInput contains transport-independent webhook settings.
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
}

// NewWebhooks constructs outgoing webhook use cases.
func NewWebhooks(repository webhookRepository, secretCipher *secrets.Cipher) *Webhooks {
	return &Webhooks{
		repository: repository,
		secrets:    secretCipher,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

// WebhookEvents returns the supported outgoing event names.
func WebhookEvents() []string {
	return slices.Clone(webhookEvents)
}

// Webhooks returns configured webhooks with signing secrets redacted.
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

// WebhookDeliveries returns recent outgoing delivery attempts.
func (s *Webhooks) WebhookDeliveries(ctx context.Context, limit int) ([]domain.WebhookDelivery, error) {
	return s.repository.WebhookDeliveries(ctx, limit)
}

// SaveWebhook validates, protects, and persists one webhook configuration.
func (s *Webhooks) SaveWebhook(ctx context.Context, id int64, input WebhookInput) (domain.Webhook, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	events := normalizeWebhookEvents(input.Events)

	if err := validateWebhookInput(input, events); err != nil {
		return domain.Webhook{}, err
	}

	secretValue, err := s.resolveWebhookSecret(ctx, id, input)
	if err != nil {
		return domain.Webhook{}, err
	}

	item, err := s.repository.SaveWebhook(ctx, id, domain.Webhook{
		Name:    input.Name,
		URL:     input.URL,
		Events:  events,
		Secret:  secretValue,
		Enabled: input.Enabled,
	})
	if err != nil {
		return domain.Webhook{}, err
	}

	item.SecretConfigured = item.Secret != ""
	item.Secret = ""

	return item, nil
}

// validateWebhookInput returns all user-correctable webhook configuration failures.
func validateWebhookInput(input WebhookInput, events []string) error {
	validation := &ValidationError{}

	if input.Name == "" {
		validation.Fields = append(validation.Fields, FieldError{Field: "name", Message: "A webhook name is required."})
	}

	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		validation.Fields = append(validation.Fields, FieldError{Field: "url", Message: "Enter an absolute HTTP or HTTPS URL."})
	}

	if len(events) == 0 {
		validation.Fields = append(validation.Fields, FieldError{Field: "events", Message: "Choose at least one event."})
	}

	if len(validation.Fields) == 0 {
		return nil
	}

	return validation
}

// resolveWebhookSecret preserves, clears, or encrypts a webhook signing secret.
func (s *Webhooks) resolveWebhookSecret(ctx context.Context, id int64, input WebhookInput) (string, error) {
	secretValue := ""

	if id != 0 && !input.ClearSecret {
		existing, err := s.repository.Webhook(ctx, id)
		if err != nil {
			return "", err
		}
		secretValue = existing.Secret
	}

	if strings.TrimSpace(input.Secret) == "" {
		return secretValue, nil
	}

	if s.secrets == nil || !s.secrets.Configured() {
		return "", newValidationError("secret", "Configure LORE__ENCRYPTION_KEY before saving a signing secret.")
	}

	return s.secrets.Encrypt(input.Secret)
}

// DeleteWebhook removes one outgoing webhook configuration.
func (s *Webhooks) DeleteWebhook(ctx context.Context, id int64) error {
	return s.repository.DeleteWebhook(ctx, id)
}

// TestWebhook sends a diagnostic event to one webhook regardless of its filters.
func (s *Webhooks) TestWebhook(ctx context.Context, id int64) error {
	item, err := s.repository.Webhook(ctx, id)
	if err != nil {
		return err
	}

	event := OutgoingEvent{
		Event:      "webhook.test",
		ObjectType: "webhook",
		ObjectKey:  item.Name,
		Detail:     "Test delivery from Lore",
		OccurredAt: time.Now().UTC(),
	}

	return s.deliver(ctx, item, event)
}

// Emit delivers one committed application event to every matching webhook.
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

		deliveryErr := s.deliver(ctx, item, event)
		if deliveryErr == nil {
			continue
		}

		combined = errors.Join(combined, deliveryErr)
	}

	return combined
}

// deliver sends one signed webhook request and records its outcome.
func (s *Webhooks) deliver(ctx context.Context, item domain.Webhook, event OutgoingEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	request, err := webhookRequest(ctx, item, event, body)
	if err != nil {
		return err
	}

	if err := s.signWebhookRequest(request, item.Secret, body); err != nil {
		s.recordWebhookDelivery(ctx, item.ID, event.Event, 0, err.Error())
		return err
	}

	response, err := s.client.Do(request)
	if err != nil {
		deliveryErr := fmt.Errorf("deliver webhook %q: %w", item.Name, err)
		s.recordWebhookDelivery(ctx, item.ID, event.Event, 0, err.Error())
		return deliveryErr
	}
	defer response.Body.Close() // nolint:errcheck

	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))

	message, deliveryErr := webhookResponseError(item.Name, response)
	recordErr := s.repository.AddWebhookDelivery(ctx, item.ID, event.Event, response.StatusCode, message)
	if deliveryErr != nil {
		return deliveryErr
	}

	return recordErr
}

// webhookRequest constructs the stable HTTP request envelope for an outgoing event.
func webhookRequest(ctx context.Context, item domain.Webhook, event OutgoingEvent, body []byte) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, item.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Lore-Webhook/1")
	request.Header.Set("X-Lore-Event", event.Event)

	return request, nil
}

// signWebhookRequest adds an HMAC signature when the webhook has a signing secret.
func (s *Webhooks) signWebhookRequest(request *http.Request, encryptedSecret string, body []byte) error {
	if encryptedSecret == "" {
		return nil
	}

	if s.secrets == nil || !s.secrets.Configured() {
		return secrets.ErrNotConfigured
	}

	plain, err := s.secrets.Decrypt(encryptedSecret)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, []byte(plain))
	_, _ = mac.Write(body)
	request.Header.Set("X-Lore-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))

	return nil
}

// webhookResponseError classifies non-successful HTTP responses for delivery history.
func webhookResponseError(name string, response *http.Response) (string, error) {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return "", nil
	}

	return response.Status, fmt.Errorf("webhook %q returned %s", name, response.Status)
}

// recordWebhookDelivery records a failed delivery when the primary error must be preserved.
func (s *Webhooks) recordWebhookDelivery(ctx context.Context, webhookID int64, event string, statusCode int, message string) {
	_ = s.repository.AddWebhookDelivery(ctx, webhookID, event, statusCode, message)
}

// normalizeWebhookEvents filters, deduplicates, and preserves supported events.
func normalizeWebhookEvents(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}

	for _, value := range values {
		value = strings.TrimSpace(value)
		if !slices.Contains(webhookEvents, value) {
			continue
		}
		if seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)
	}

	return result
}

package service

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type webhookRepositoryStub struct {
	items      []domain.Webhook
	deliveries []domain.WebhookDelivery
	saved      domain.Webhook
}

func (r *webhookRepositoryStub) Webhooks(context.Context) ([]domain.Webhook, error) {
	return r.items, nil
}
func (r *webhookRepositoryStub) Webhook(_ context.Context, id int64) (domain.Webhook, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Webhook{}, domain.ErrNotFound
}
func (r *webhookRepositoryStub) SaveWebhook(_ context.Context, _ int64, item domain.Webhook) (domain.Webhook, error) {
	r.saved = item
	item.ID = 1
	return item, nil
}
func (*webhookRepositoryStub) DeleteWebhook(context.Context, int64) error { return nil }
func (r *webhookRepositoryStub) AddWebhookDelivery(_ context.Context, id int64, event string, status int, message string) error {
	r.deliveries = append(r.deliveries, domain.WebhookDelivery{WebhookID: id, Event: event, StatusCode: status, Error: message})
	return nil
}
func (r *webhookRepositoryStub) WebhookDeliveries(context.Context, int) ([]domain.WebhookDelivery, error) {
	return r.deliveries, nil
}

func testSecretCipher(t *testing.T) *secrets.Cipher {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	cipher, err := secrets.New(key)
	require.NoError(t, err)
	return cipher
}

func TestWebhooks(t *testing.T) {
	t.Parallel()

	t.Run("signs matching event", func(t *testing.T) {
		t.Parallel()
		var gotEvent, gotSignature string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotEvent = r.Header.Get("X-Lore-Event")
			gotSignature = r.Header.Get("X-Lore-Signature")
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		cipher := testSecretCipher(t)
		encrypted, err := cipher.Encrypt("hook-secret")
		require.NoError(t, err)
		repository := &webhookRepositoryStub{items: []domain.Webhook{{ID: 1, Name: "build", URL: server.URL, Events: []string{"page.updated"}, Secret: encrypted, Enabled: true}}}
		err = NewWebhooks(repository, cipher, slog.Default()).Emit(context.Background(), OutgoingEvent{Event: "page.updated", ObjectType: "page", ObjectKey: "guide"})
		require.NoError(t, err)
		assert.Equal(t, "page.updated", gotEvent)
		assert.Contains(t, gotSignature, "sha256=")
		require.Len(t, repository.deliveries, 1)
		assert.Equal(t, http.StatusNoContent, repository.deliveries[0].StatusCode)
	})

	t.Run("requires encryption key for signing secret", func(t *testing.T) {
		t.Parallel()
		repository := &webhookRepositoryStub{}
		_, err := NewWebhooks(repository, &secrets.Cipher{}, slog.Default()).SaveWebhook(context.Background(), 0, WebhookInput{Name: "hook", URL: "https://example.test/hook", Events: []string{"page.updated"}, Secret: "secret", Enabled: true})
		validation, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Equal(t, "secret", validation.Fields[0].Field)
	})
}

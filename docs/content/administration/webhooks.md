# Webhooks

Administrators can configure outgoing HTTP webhooks under **Administration → Webhooks**. Each webhook has a name, an absolute HTTP(S) endpoint, an enabled flag, a JSON payload template, optional request headers, retry settings, and an event allow-list.

## Payload templates

Lore uses Notifykit to render the request body with Go `text/template`. The rendered result must be valid JSON. Notifykit's default template helpers are available, including `json`; use it whenever a template value is inserted into JSON.

The template context contains a stable event input and Lore payload data:

```text
.Input.Event
.Payload.ActorID
.Payload.ObjectType
.Payload.ObjectKey
.Payload.Detail
.Payload.OccurredAt
.Payload.URL
.Receiver
.Title
```

For example, a Slack-style payload can be configured as:

```gotemplate
{
  "text": {{ print .Input.Event ": " .Payload.Detail | json }},
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": {{ print "*Page:* " .Payload.ObjectKey "\n*URL:* " .Payload.URL | json }}
      }
    }
  ]
}
```

`Payload.URL` is populated for page events when `LORE__PUBLIC_URL` is configured. **Send test** renders the same configured template against a `webhook.test` event, regardless of the webhook's event allow-list.

## Request headers

Webhooks can send arbitrary application-level HTTP request headers, for example `Authorization`, `X-API-Key`, or receiver-specific metadata. Lore always sends `X-Lore-Event` and a `Lore-Webhook/1` user agent unless the webhook configuration overrides them. Transport-controlled headers such as `Host`, `Content-Length`, and `Content-Type` cannot be configured; the request body is JSON and Notifykit supplies its JSON content type.

Headers containing credentials can be marked sensitive. Sensitive values are encrypted in PostgreSQL using `LORE__ENCRYPTION_KEY`, masked in the administration UI, and decrypted only for delivery or an explicit administrator reveal action.

## Retries

Retries are optional per webhook and disabled by default. When enabled, Lore explicitly uses Notifykit's `DefaultRetryPolicy`. The policy retries classified network and timeout failures, HTTP `408`, HTTP `429`, and HTTP `5xx` responses. Other client responses and permanent configuration or template failures are not retried.

The retry count is the number of attempts **after** the initial request. Backoff is exponential from the configured initial value up to the configured maximum. Jitter can randomize the local wait, and a receiver-provided HTTP `Retry-After` value is honored as a minimum delay. Disabling retries makes exactly one delivery attempt.

## Delivery behavior

Page creates/updates/moves/deletes, review requests/updates/cancellations/decisions, revision restores, comments, imports, and bulk operations emit events after the primary mutation commits. Webhook delivery remains a best-effort side effect: an unavailable endpoint does not roll back a successful page change.

Lore records the final HTTP status or delivery error together with the number of attempts in the recent-deliveries list. The webhook HTTP client uses a five-second timeout for each attempt.

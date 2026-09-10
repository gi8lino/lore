# Webhooks

Administrators can configure outgoing HTTP webhooks under **Administration → Webhooks**. Each destination has a name, absolute HTTP(S) endpoint, an enabled flag, an event allow-list, and an optional signing secret.

When a signing secret is set, Lore encrypts it in PostgreSQL with `LORE__ENCRYPTION_KEY`. The plaintext is not returned to the administration UI. Deliveries include `X-Lore-Event` and, when signed, `X-Lore-Signature: sha256=<hex HMAC>` calculated over the exact JSON request body.

The JSON payload contains the event name, actor identifier, object type/key, detail, and UTC occurrence time. Page creates/updates/moves/deletes, review activity, revision restores, comments, imports, and bulk operations emit events after the primary mutation commits.

Webhook delivery is a best-effort side effect: an unavailable endpoint does not roll back a successful page change. Lore records the HTTP status or delivery error in the recent-deliveries list. The HTTP client uses a five-second timeout. **Send test** emits a `webhook.test` payload to one configured destination regardless of its event allow-list.

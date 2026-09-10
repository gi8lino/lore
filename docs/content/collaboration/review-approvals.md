# Review approvals

Lore adds a lightweight approval step on top of page lifecycle and immutable revision history. It does not create branches or merge copies of a page.

## Request a review

Editors can request review for the current revision from the page action bar. A request can target one or more people with `@username`, a collaboration group, or both. Only enabled `editor` and `admin` accounts can be selected as individual reviewers. Any one assigned reviewer may make the decision. Administrators can always review as a recovery path.

The page owner group is selected by default when one exists. If a request has no explicit people or group, Lore falls back to the owner group and then to administrators. Review assignment does not bypass page access rules: a reviewer still needs normal access to read the page.

Opening a request records the exact current revision and moves the page lifecycle to `draft`. The request note is optional and is intended to describe what the reviewer should focus on.

## Edit or cancel a pending request

While a request is pending, its requester or an administrator can edit the assigned people, reviewer group, and request note. These changes keep the same review request and the same revision number; editing a request never changes what content is being reviewed.

The requester or an administrator can also cancel a pending request. A canceled request remains in audit history and cannot be reopened or rewritten. When the page has not changed since the request was opened, canceling restores the lifecycle state that the page had before the request.

## Decide a review

An assigned reviewer can **Approve** or **Request changes**. Administrators can always make either decision.

Approval changes the page to `verified`, records the review timestamp, and closes the request. Completed approval UI disappears from the page; the immutable decision remains available through revision/audit history.

Requesting changes closes that review decision and keeps the page as a draft. The author sees the reviewer note as an actionable page message. Editing the page supersedes that request automatically; after the new revision is saved, the author can request a fresh review.

If a page changes while a request is still pending, the old request is superseded and can no longer be approved. A reviewer submitting an already-open stale form receives a conflict instead of approving a different revision.

Review requests, request edits, cancellations, decisions, and normal page changes are recorded in audit history and can be delivered through page watches and outgoing webhooks.

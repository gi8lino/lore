# Discussions and notifications

Administrators can enable or disable page discussions globally.

When discussions are enabled, readers can add comments to a page. Comments may include an anchor copied from selected page text. Editors can resolve or reopen discussion items.

Lore also has a lightweight per-user notification inbox in the top bar. The bell shows the unread count and the newest notifications. Opening an item marks it read before Lore redirects to the notification's stored local destination; **Mark all read** clears the unread state for the complete inbox. The notification API and read/open responses are private and non-cacheable.

Mention processing creates notification items for distinct `@username` references in page content and comments. Lore ignores mentions of the acting user and usernames that do not resolve to an account. Notification destinations are restricted to local application paths, so an inbox item cannot redirect the browser to an external site.

The page view keeps discussion state separate from immutable page revision history: comments do not rewrite Markdown revisions.

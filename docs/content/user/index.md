# User experience

The full Lore server gives each authenticated user a personal dashboard, presentation preferences, saved searches, favorites, recent-page shortcuts, private editor drafts, notifications, and personal access tokens.

- [Dashboard](dashboard.md)
- [Settings and preferences](settings.md)

These features depend on a Lore account and PostgreSQL. Filesystem static sites intentionally omit them and keep only local read-only presentation state such as temporarily hiding the navigation or page contents.

{{subpages}}

## Offline pages

When service workers are available, Lore can keep previously visited pages in a private browser cache. A tab must first identify its signed-in user online. Only responses carrying that same server-authenticated identity are cached; login redirects are excluded. Signing out or switching accounts clears private caches. After the browser restarts its service worker, reconnect online before relying on offline pages. Offline copies are browser-local snapshots and cannot reflect server-side revocation until the browser reconnects.

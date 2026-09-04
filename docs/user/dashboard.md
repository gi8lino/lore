# Dashboard

The server home page is a personalized documentation dashboard.

It combines:

- favorite pages for the current user;
- recently updated pages;
- recently viewed pages;
- the most viewed active pages;
- the current user's recent edits;
- private server drafts for `admin` and `editor` accounts.

Favorites can also appear as pinned pages in the sidebar. Recently viewed pages can be enabled as a separate sidebar section in user preferences.

Opening a page records a view for the authenticated user and contributes to recent-view and popularity features. The static filesystem site has no accounts or server-side view tracking, so it does not generate a personalized dashboard.

# Review approvals

Lore adds a lightweight approval step on top of page lifecycle and revision history. It does not create branches or merge copies of a page.

An editor with edit access can request review for the current revision. Lore records that exact revision number, moves the page lifecycle to `draft`, and notifies members of the configured owner group. If the page has no owner group, administrators receive the request instead.

Members of the owner group who also have the `editor` role can approve the requested revision or request changes. Administrators can always review. An approval changes the page to `verified` and records the review timestamp. A requested-changes decision leaves the page as a draft and notifies the requester.

If the page is edited after review was requested, the pending request becomes stale. Lore refuses to approve it; an editor must request a new review so the approval always refers to known content. Review requests and decisions also appear in audit history and page-watch notifications.

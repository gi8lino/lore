# Page watches

Authenticated users can subscribe to changes from the page actions. A **page** watch follows the exact page; a **subtree** watch follows the selected path and every page below it.

Lore adds inbox notifications when another user creates or edits matching content, moves or deletes a watched page, completes a documentation review, restores a revision, or adds a discussion comment. The actor never receives a notification for their own change, and overlapping watches produce only one notification for an event.

Watches are path based. A subtree subscription such as `platform/kubernetes` also matches `platform/kubernetes/ingress`. A move notification links to the new path so watchers can decide whether to subscribe there.

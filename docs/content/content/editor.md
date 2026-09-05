# Editor and drafts

The editor is available to `admin` and `editor` accounts. It works with Markdown source and server-rendered preview so preview behavior matches persisted page rendering.

## Page fields

A save can include:

- path and title;
- optional Lucide icon and content-language override;
- Markdown body and revision message;
- tags and collaboration groups;
- lifecycle status;
- owner group and review interval;
- deprecated replacement target;
- structured key/value properties.

## Draft protection

Lore has two draft mechanisms with different purposes.

Browser autosave protects unsaved form state in the current browser. Private server drafts persist editor state per user in PostgreSQL so unfinished work can resume on another device. Saving or explicitly discarding removes the private server draft. Server drafts track the page revision they started from and can indicate that the persisted page changed underneath them.

A page whose lifecycle status is `draft` is different: it is a real persisted page visible through normal page/history behavior. See [Page lifecycle](lifecycle.md).

## Revision history

Every persisted page update creates immutable revision history. The page view shows the newest revision summary and can load the full history. Editors can restore an older revision by creating a new revision from its Markdown rather than mutating history in place.

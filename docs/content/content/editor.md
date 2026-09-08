# Editor and drafts

The editor is available to `admin` and `editor` accounts. It works with Markdown source and server-rendered preview so preview behavior matches persisted page rendering.

![Lore Markdown editor showing this page in split view](../assets/screenshots/editor.png)

The editor opens in your last **Write** or **Split** view. **Preview** is temporary and never becomes the starting view.

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

## Quick insert

The Markdown editor provides the same reusable insert actions from the keyboard and toolbar:

- type `@` to search for and insert a user mention;
- type `{{` to search stored variables and insert the canonical `{{var:name}}` macro;
- use **Insert → Mention** or **Insert → Variable** for the same pickers;
- type `/` at the start of a line to open editor commands. `/mention` and `/variable` open the corresponding pickers, while
  stored variables and snippets also appear as direct slash-command results.

Autocomplete is suppressed inside fenced code blocks where Lore keeps knowledge macros literal.

The Markdown source editor always disables coding ligatures so operator sequences remain visually literal while editing. The **Coding ligatures** rendering setting shows operator ligatures in ordinary text, inline code, and fenced code in the preview and rendered page. When enabled alongside **Typographer**, ASCII operators such as `-->`, `<<`, and `>>` are preserved instead of being converted to dashes or quotation marks.

## Draft protection

Lore has two draft mechanisms with different purposes.

Browser autosave protects unsaved form state in the current browser. Private server drafts persist editor state per user in PostgreSQL so unfinished work can resume on another device. Saving or explicitly discarding removes the private server draft. Server drafts track the page revision they started from and can indicate that the persisted page changed underneath them.

A page whose lifecycle status is `draft` is different: it is a real persisted page visible through normal page/history behavior. See [Page lifecycle](lifecycle.md).

## Revision history

Every persisted page update creates immutable revision history. The page view shows the newest revision summary and can load the full history. Editors can restore an older revision by creating a new revision from its Markdown rather than mutating history in place.

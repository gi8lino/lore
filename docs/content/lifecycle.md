# Page lifecycle

Every page has one of four lifecycle states:

| Status | Meaning in the interface |
| --- | --- |
| `draft` | Persisted documentation that has not been verified yet. |
| `verified` | Normal active documentation. |
| `deprecated` | Content retained with an optional replacement target. |
| `archived` | Content retained for historical context. |

A lifecycle `draft` is a real page and is not the same as a user's private editor draft.

Deprecated pages can point to a replacement path. The page view shows a lifecycle banner so readers can follow the replacement.

Administrators can apply status changes in bulk. Documentation health also reports draft and deprecated pages.

Deleted pages are not a lifecycle status. Deletion moves a page into the administrator recycle bin, where it can be restored or permanently deleted.

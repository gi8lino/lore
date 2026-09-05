# Groups and roles

Lore users have one role: `admin`, `editor`, or `viewer`.

Route authorization uses those roles to protect administration, editing, media mutation, and destructive operations. The service layer also enforces page-specific mutation policy so the same rules are not limited to HTTP callers.

Administrators manage named groups and group membership. Pages can be assigned to groups, and editors may only assign groups that are available to their account. A separate owner group can represent documentation ownership for review workflows.

OIDC can synchronize explicitly mapped external groups into Lore groups. See [OIDC](../authentication/oidc.md).

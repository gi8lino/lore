# Organizing pages

Lore derives navigation from page paths rather than maintaining a separate page tree.

```text
platform
platform/kubernetes
platform/kubernetes/ingress
```

A path segment can exist as a synthetic navigation folder even when no page exists at that exact path. Administrators can assign Lucide icons to navigation paths.

## Tags

Tags provide a cross-cutting taxonomy and are searchable with `tag:` filters. The administrator tag view shows usage counts and can delete tags.

## Groups

Pages can be assigned collaboration groups. Group membership controls which groups an editor is allowed to assign. Groups are also searchable. See [Groups and roles](../collaboration/groups.md).

## Ownership and review metadata

A page can name an owner group and a review interval. Explicit review actions update the review timestamp, and documentation health can surface pages whose review is due.

## Structured properties

Pages can store searchable key/value properties. Search uses `property:key=value` to match a property key and value.

## Moving pages

Lore can move one page or a subtree. Move options can include moving children, updating incoming wiki links, and retaining aliases. Aliases let old page paths resolve to the active path.

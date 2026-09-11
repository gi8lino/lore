# Settings and preferences

The **Settings** page shows the current account identity, role, and group memberships and provides user-specific presentation and access controls.

## Presentation

Users can select a theme, choose a personal typography size (Application default, Compact, Standard, or Large), show or hide the page table of contents, choose a persistent desktop navigation style (Sidebar, Top bar, or Page tree), choose comfortable or compact navigation density, and set the desktop sidebar width. Sidebar width offers Narrow, Standard, and Wide presets while direct edge dragging can store a custom width between 220 and 420 pixels.

The administrator chooses the application typography default, which is Compact on new installations. Users who leave **Application default** selected follow that setting; a personal Compact, Standard, or Large choice overrides it for that account. The typography preset scales page titles, Markdown headings, and reading text together.

Navigation preferences also control indentation guides, whether expanded folders are remembered, whether favorite pages are pinned above the tree, whether recently viewed pages are shown, and whether folder page counts are displayed.

## Saved searches

Any server search query can be stored with a name. Saved searches can be pinned into the sidebar, which makes filtered documentation collections available without retyping the query.

## Personal access tokens

Users can issue named personal access tokens with an optional expiration date. Token secrets are shown only at creation time and inherit the current user's Lore role. See [Personal access tokens](../api/tokens.md).

## Uploaded images

Editors can review images they uploaded from their settings area. The list starts with the 30 newest uploads, supports filename search, and loads older matches in additional batches. Only unused owned images can be deleted there.

Administrators can manage uploads across every user under **Administration → Images**. That view also starts with the 30 newest images, supports search by filename or uploader, and loads additional matches on demand.

Static sites have no account settings. Their read-only navigation/page-content toggles use browser-local state only and never call a Lore server.

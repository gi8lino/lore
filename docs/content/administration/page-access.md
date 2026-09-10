# Page access

Lore can restrict a page path and its descendants to selected collaboration groups without introducing a separate space or collection hierarchy.

Administrators manage rules under **Administration → Page access**. A rule grants one group either **View** or **View and edit** access to a path. Multiple rules on the same path form an allow-list. Edit access also grants view access.

Rules inherit down the page tree. When a descendant has its own rules, that nearest rule set replaces the inherited one. Paths without any matching rule remain available according to the normal Lore role model. Administrators always bypass page-path restrictions so an installation cannot be locked permanently.

Page access is applied to normal page reads and mutations, navigation, dashboard collections, search, the knowledge graph, page reports, includes, revisions, comments, and exports. Restricted pages are omitted from the public sitemap. Authentication remains required for the normal Lore server even when a path has no explicit access rule.

Access rules protect page content. Uploaded media and attachments retain Lore's existing authenticated media authorization and are not separate ACL objects; avoid sharing a raw media URL as an independent secret.

# Documentation health

Lore computes documentation-quality findings for administrators. The health view groups pages into actionable categories:

- broken wiki links;
- orphan pages;
- untagged pages;
- pages without an icon;
- stale pages;
- pages whose configured review is due;
- lifecycle `draft` pages;
- deprecated pages.

The stale-page view uses an age cutoff supplied by the handler; the administration page currently checks against roughly six months.

These checks use the PostgreSQL-backed page inventory and link graph. They are a server feature, not part of the read-only filesystem static build.

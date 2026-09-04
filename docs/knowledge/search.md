# Search

The server uses PostgreSQL full-text search plus field filters. Free-text terms are passed through PostgreSQL `websearch_to_tsquery` and results are ranked before update time.

Supported filters are:

```text
tag:name
group:name
title:text
namespace:path
author:name
status:draft|verified|deprecated|archived
owner:group
property:key=value
```

Quoted filter values can contain spaces, for example:

```text
group:"Platform Engineering" status:verified postgres
```

Multiple filters are combined with the free-text terms. Saved searches let users name queries and optionally pin them into the sidebar.

## Static-site search

Filesystem static mode cannot use PostgreSQL. The builder writes `search-index.json` and the read-only TypeScript frontend performs lightweight title/body matching in the browser. Server field filters are therefore not available in static mode.

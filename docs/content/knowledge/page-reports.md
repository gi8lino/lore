# Dynamic page reports

Lore can render a live collection of pages from the normal server search language. The report is evaluated when the page is rendered, so changes to matching page metadata appear without editing the report page.

Use a standalone `{{pages ...}}` function:

```text
{{pages query="tag:service status:verified"}}
```

The required `query` option accepts the same filters as normal search. Optional settings control the columns, presentation, ordering, and result limit:

```text
{{pages query="owner:\"Platform\"" columns="title,property:version,status,updated" view=table sort=title limit=50}}
```

Supported columns are `title`, `path`, `status`, `owner`, `updated`, `author`, `tags`, `views`, and `property:<key>`. Supported views are `table`, `list`, and `cards`. Sorting can use `relevance`, `updated`, `title`, or `path`; the limit may be between 1 and 100.

Page reports are a server knowledge feature because they execute PostgreSQL-backed search queries. Filesystem static builds leave the source invocation unchanged rather than querying a database.

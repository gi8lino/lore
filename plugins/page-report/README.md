# Page Report

Page Report adds the `{{pages ...}}` macro for rendering live lists of pages that match a Lore search query.

## Usage

```text
{{pages query="tag:service status:verified"}}
```

A more detailed report can select columns, presentation, sorting, and a result limit:

```text
{{pages query="owner:\"Platform\"" columns="title,property:version,status,updated" view=table sort=title limit=50}}
```

`query` is required. `columns` defaults to `title,status,owner,updated`; supported columns are `title`, `path`, `status`, `owner`, `updated`, `author`, `tags`, `views`, and `property:<key>`. `view` supports `table`, `list`, and `cards`. `sort` supports `relevance`, `updated`, `title`, and `path`. `limit` defaults to 20 and accepts values from 1 to 100.

## Permissions

This plugin requests `pages:read` to search pages and load the metadata needed for report results.

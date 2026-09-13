CREATE TABLE plugin_installations (
    plugin_id text PRIMARY KEY CHECK (octet_length(plugin_id) BETWEEN 1 AND 128),
    source text NOT NULL CHECK (source IN ('bundled', 'installed')),
    enabled boolean NOT NULL,
    package bytea,
    CHECK ((source = 'bundled' AND package IS NULL) OR
           (source = 'installed' AND package IS NOT NULL AND octet_length(package) BETWEEN 1 AND 16777216))
);

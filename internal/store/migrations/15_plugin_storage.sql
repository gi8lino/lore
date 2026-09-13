CREATE TABLE plugin_values (
    plugin_id text NOT NULL,
    namespace text NOT NULL CHECK (namespace IN ('settings', 'data')),
    key text NOT NULL CHECK (octet_length(key) BETWEEN 1 AND 256),
    value bytea NOT NULL CHECK (octet_length(value) <= 65536),
    PRIMARY KEY (plugin_id, namespace, key)
);

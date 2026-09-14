INSERT INTO plugin_values (plugin_id, namespace, key, value)
SELECT
  'io.lore.snippets',
  'data',
  'r:snippets:' || lower(name),
  convert_to(json_build_object(
    'name', name,
    'description', description,
    'content', content
  )::text, 'UTF8')
FROM knowledge_snippets
WHERE kind = 'snippet'
ON CONFLICT (plugin_id, namespace, key) DO NOTHING;

DROP TABLE knowledge_snippets;

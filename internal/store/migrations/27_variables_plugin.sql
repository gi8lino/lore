INSERT INTO plugin_values (plugin_id, namespace, key, value)
SELECT
  'io.lore.variables',
  'data',
  'r:variables:' || lower(name),
  convert_to(json_build_object(
    'name', name,
    'description', description,
    'content', content
  )::text, 'UTF8')
FROM knowledge_snippets
WHERE kind = 'variable'
ON CONFLICT (plugin_id, namespace, key) DO NOTHING;

DELETE FROM knowledge_snippets
WHERE kind = 'variable';

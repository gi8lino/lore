UPDATE navigation_icons
SET icon = icon || '-lucide'
WHERE icon <> '';

UPDATE application_settings
SET external_links = (
  SELECT coalesce(
    jsonb_agg(
      CASE
        WHEN coalesce(link->>'icon', '') = '' THEN link
        ELSE jsonb_set(link, '{icon}', to_jsonb((link->>'icon') || '-lucide'))
      END
      ORDER BY ordinality
    ),
    '[]'::jsonb
  )
  FROM jsonb_array_elements(external_links) WITH ORDINALITY AS links(link, ordinality)
)
WHERE jsonb_array_length(external_links) > 0;

UPDATE page_drafts
SET form_values = jsonb_set(
  form_values,
  '{icon,0}',
  to_jsonb((form_values->'icon'->>0) || '-lucide')
)
WHERE jsonb_typeof(form_values->'icon') = 'array'
  AND coalesce(form_values->'icon'->>0, '') <> '';

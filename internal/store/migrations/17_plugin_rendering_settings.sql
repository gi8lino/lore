ALTER TABLE application_settings
  RENAME COLUMN render_content_language TO content_language;

ALTER TABLE application_settings
  DROP COLUMN render_callouts,
  DROP COLUMN render_tables,
  DROP COLUMN render_table_styles,
  DROP COLUMN render_table_sorting,
  DROP COLUMN render_table_filtering,
  DROP COLUMN render_coding_ligatures,
  DROP COLUMN render_mermaid;

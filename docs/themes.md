# Themes

Lore themes are TOML files containing a `color_scheme` (`light` or `dark`) and a complete set of semantic color roles. Built-in themes include Light, Dark, and Catppuccin Mocha.

Semantic roles cover surfaces, text emphasis, borders, accents, status colors, and text selection. Components consume CSS custom properties derived from those roles rather than hard-coded theme colors.

The normal server can load additional theme files from `LORE_THEME_DIRECTORY`; external files with the same title override embedded themes.

Static site mode uses the theme named in `lore-site.toml`. The generated page embeds the theme catalog and uses the same Lore CSS, so Markdown tables, callouts, code, and navigation retain theme-aware styling.

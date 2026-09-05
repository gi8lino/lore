# Themes

Lore themes are TOML files containing a `color_scheme` (`light` or `dark`) and a complete set of semantic color roles. Components consume CSS custom properties derived from those roles rather than hard-coded theme colors.

## Built-in themes

Lore includes a broad set of light and dark themes:

- **Lore:** Light, Dark
- **Catppuccin:** Latte, Frappe, Macchiato, Mocha
- **Dracula**
- **Nord**
- **Gruvbox:** Dark, Light
- **Tokyo Night**
- **Solarized:** Dark, Light
- **One Dark**
- **Rose Pine:** Rose Pine, Rose Pine Dawn

The theme title is derived from the TOML filename. Users can select any available theme from their personal settings.

## Custom themes

The normal server can load additional theme files from `LORE__THEME_DIRECTORY`. External files with the same filename as an embedded theme override that built-in theme, so deployments can tune the bundled palettes without changing Lore itself.

Every theme must provide the complete semantic palette: application surfaces, primary and secondary text, borders, accents, success/warning/error colors, and text-selection colors.

Static site mode uses the theme named in `docs/lore-site.toml`. The generated page embeds the theme catalog and uses the same Lore CSS, so Markdown tables, callouts, code, and navigation retain theme-aware styling.

// Theme selection and persistence.

interface ThemeDefinition {
  title: string;
  color_scheme: string;
  colors: Record<string, string>;
}

function parseThemeCatalog(source: string): ThemeDefinition[] {
  try {
    const value: unknown = JSON.parse(source);
    if (!Array.isArray(value)) return [];
    return value.filter((candidate): candidate is ThemeDefinition => {
      if (typeof candidate !== "object" || candidate === null) return false;
      const theme = candidate as Partial<ThemeDefinition>;
      return (
        typeof theme.title === "string" &&
        typeof theme.color_scheme === "string" &&
        typeof theme.colors === "object" &&
        theme.colors !== null
      );
    });
  } catch {
    return [];
  }
}

const themeCatalog = parseThemeCatalog(
  document.querySelector("#lore-themes")?.textContent ?? "[]",
);
const themeSelect = document.querySelector<HTMLSelectElement>(
  "[data-theme-select]",
);
const activeTheme = document.documentElement.dataset.theme || "Light";

// Finds a configured theme by name.
function findTheme(title: string): ThemeDefinition | undefined {
  const normalized = title.toLocaleLowerCase();
  return themeCatalog.find(
    (candidate) => candidate.title.toLocaleLowerCase() === normalized,
  );
}

// Applies theme.
function applyTheme(title: string): void {
  const theme = findTheme(title) ?? findTheme(activeTheme) ?? themeCatalog[0];
  if (!theme) return;

  for (const [key, value] of Object.entries(theme.colors)) {
    document.documentElement.style.setProperty(
      `--${key.replaceAll("_", "-")}`,
      value,
    );
  }

  document.documentElement.style.colorScheme = theme.color_scheme;
  document.documentElement.dataset.theme = theme.title;
  if (themeSelect) themeSelect.value = theme.title;
}

// Initializes theme.
export function initTheme(): void {
  applyTheme(activeTheme);
  themeSelect?.addEventListener("change", (event: Event) => {
    const target = event.currentTarget;
    if (target instanceof HTMLSelectElement) applyTheme(target.value);
  });
}

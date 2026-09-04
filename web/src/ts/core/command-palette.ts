// Global command palette search and keyboard interaction.

const maxResults = 7;

interface CommandPage {
  slug: string;
  title: string;
}

function isCommandPage(value: unknown): value is CommandPage {
  if (typeof value !== "object" || value === null) return false;
  const page = value as Partial<CommandPage>;
  return typeof page.slug === "string" && typeof page.title === "string";
}

// Wires command-palette search, navigation, and actions.
function setupCommandPalette(dialog: HTMLDialogElement): void {
  const input = dialog.querySelector<HTMLInputElement>(
    "[data-command-palette-input]",
  );
  const results = dialog.querySelector<HTMLElement>(
    "[data-command-palette-results]",
  );
  const closeButtons = dialog.querySelectorAll<HTMLButtonElement>(
    "[data-command-palette-close]",
  );
  if (!input || !results) return;
  const inputElement = input;
  const resultsElement = results;

  let pages: CommandPage[] = [];
  let active = -1;
  let requestID = 0;

  // Returns the currently visible command-palette options.
  function allOptions(): HTMLElement[] {
    return [
      ...dialog.querySelectorAll<HTMLElement>(
        "[data-command-option]:not([hidden])",
      ),
    ];
  }

  // Sets active.
  function setActive(index: number): void {
    const options = allOptions();
    if (!options.length) {
      active = -1;
      return;
    }
    active = ((index % options.length) + options.length) % options.length;
    options.forEach((option, optionIndex) =>
      option.setAttribute("aria-selected", String(optionIndex === active)),
    );
    options[active]?.scrollIntoView({ block: "nearest" });
  }

  // Renders pages.
  function renderPages(): void {
    resultsElement.replaceChildren();
    pages.forEach((page) => {
      const anchor = document.createElement("a");
      anchor.href = `/pages/${page.slug}`;
      anchor.dataset.commandOption = "";
      anchor.className = "command-palette-option";
      anchor.setAttribute("role", "option");
      const strong = document.createElement("strong");
      const small = document.createElement("small");
      strong.textContent = page.title;
      small.textContent = page.slug;
      anchor.append(strong, small);
      resultsElement.append(anchor);
    });
    setActive(0);
  }

  // Runs the current command-palette search.
  async function search(): Promise<void> {
    const query = inputElement.value.trim();
    if (!query) {
      pages = [];
      resultsElement.replaceChildren();
      for (const option of dialog.querySelectorAll<HTMLElement>(
        "[data-command-static]",
      ))
        option.hidden = false;
      setActive(0);
      return;
    }
    for (const option of dialog.querySelectorAll<HTMLElement>(
      "[data-command-static]",
    )) {
      option.hidden = !(option.textContent ?? "")
        .toLocaleLowerCase()
        .includes(query.toLocaleLowerCase());
    }
    const current = ++requestID;
    try {
      const response = await fetch(
        `/api/search?q=${encodeURIComponent(query)}`,
        { headers: { Accept: "application/json" } },
      );
      if (!response.ok || current !== requestID) return;
      const value: unknown = await response.json();
      pages = Array.isArray(value)
        ? value.filter(isCommandPage).slice(0, maxResults)
        : [];
      renderPages();
    } catch {
      pages = [];
      renderPages();
    }
  }

  // Opens the command palette and resets its state.
  function open(): void {
    if (!dialog.open) dialog.showModal();
    inputElement.value = "";
    pages = [];
    resultsElement.replaceChildren();
    for (const option of dialog.querySelectorAll<HTMLElement>(
      "[data-command-static]",
    ))
      option.hidden = false;
    setActive(0);
    requestAnimationFrame(() => inputElement.focus());
  }

  // Closes the command palette.
  function close(): void {
    if (dialog.open) dialog.close();
  }

  document.addEventListener("keydown", (event: KeyboardEvent) => {
    if (
      (event.ctrlKey || event.metaKey) &&
      event.key.toLocaleLowerCase() === "k" &&
      !event.shiftKey
    ) {
      event.preventDefault();
      open();
      return;
    }
    if (!dialog.open) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive(active + 1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive(active - 1);
    } else if (event.key === "Enter") {
      const option = allOptions()[active];
      if (option) {
        event.preventDefault();
        option.click();
      }
    }
  });

  inputElement.addEventListener("input", () => void search());
  closeButtons.forEach((button) => button.addEventListener("click", close));
  dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === dialog) close();
  });
}

// Initializes command palette.
export function initCommandPalette(): void {
  const dialog = document.querySelector<HTMLDialogElement>(
    "[data-command-palette]",
  );
  if (dialog) setupCommandPalette(dialog);
}

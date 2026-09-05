// Reusable Lucide icon picker behavior.

interface IconOption {
  name: string;
  label: string;
  svg: string;
}

interface IconPage {
  items: IconOption[];
  has_more: boolean;
}

function isIconPage(value: unknown): value is IconPage {
  if (typeof value !== "object" || value === null) return false;

  const page = value as Partial<IconPage>;

  return Array.isArray(page.items) && typeof page.has_more === "boolean";
}

export function setupIconPicker(dialog: HTMLDialogElement): void {
  const search = dialog.querySelector<HTMLInputElement>(
    "[data-icon-picker-search]",
  );
  const grid = dialog.querySelector<HTMLElement>("[data-icon-picker-grid]");
  const close = dialog.querySelector<HTMLButtonElement>(
    "[data-icon-picker-close]",
  );
  const empty = dialog.querySelector<HTMLElement>("[data-icon-picker-empty]");
  if (!search || !grid || !close || !empty) return;

  const searchInput = search;
  const iconGrid = grid;
  const emptyState = empty;

  const iconsURL = dialog.dataset.iconsUrl || "/api/icons";
  let activeOwner: HTMLElement | null = null;
  let requestNumber = 0;
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  let currentQuery = "";
  let nextOffset = 0;
  let hasMore = false;
  let loading = false;

  // Returns the current icon owner value.
  function ownerValue(): string {
    return (
      activeOwner?.querySelector<HTMLInputElement>("[data-icon-picker-value]")
        ?.value || ""
    );
  }

  // Chooses icon.
  function chooseIcon(name: string, svg: string): void {
    if (!activeOwner) return;

    const value = activeOwner.querySelector<HTMLInputElement>(
      "[data-icon-picker-value]",
    );
    const preview = activeOwner.querySelector<HTMLElement>(
      "[data-icon-picker-preview]",
    );
    const label = activeOwner.querySelector<HTMLElement>(
      "[data-icon-picker-label]",
    );

    if (value) {
      value.value = name;
      value.dispatchEvent(new Event("input", { bubbles: true }));
    }
    if (label) label.textContent = name || "No icon";
    if (preview) {
      preview.replaceChildren();
      if (svg) preview.insertAdjacentHTML("afterbegin", svg);
      else if (preview.dataset.iconPickerEmpty === "plus")
        preview.insertAdjacentHTML(
          "afterbegin",
          '<svg class="lucide-icon" width="20" height="20" viewBox="0 0 24 24" ' +
            'fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" ' +
            'stroke-linejoin="round" aria-hidden="true">' +
            '<path d="M12 5v14M5 12h14"></path></svg>',
        );
    }

    closePicker();
  }

  // Builds one icon-picker option button.
  function optionButton({ name, label, svg }: IconOption): HTMLButtonElement {
    const button = document.createElement("button");

    button.className = "icon-picker-option";
    button.type = "button";
    button.dataset.iconName = name;

    const glyph = document.createElement("span");

    glyph.className = `icon-picker-glyph${name ? "" : " icon-picker-empty"}`;

    if (svg) glyph.insertAdjacentHTML("afterbegin", svg);
    else glyph.textContent = "—";

    const text = document.createElement("span");
    const strong = document.createElement("strong");

    strong.textContent = label;

    const small = document.createElement("small");

    small.textContent = name || "None";
    text.append(strong, small);
    button.append(glyph, text);
    button.addEventListener("click", () => chooseIcon(name, svg));
    return button;
  }

  // Loads options.
  async function loadOptions(
    query: string,
    { append = false }: { append?: boolean } = {},
  ): Promise<void> {
    query = query.trim();
    if (append && (loading || !hasMore || query !== currentQuery)) return;

    if (!append) {
      currentQuery = query;
      nextOffset = 0;
      hasMore = true;
      iconGrid.replaceChildren();
      emptyState.hidden = true;
    }

    const currentRequest = append ? requestNumber : ++requestNumber;
    const offset = append ? nextOffset : 0;

    loading = true;
    iconGrid.setAttribute("aria-busy", "true");

    try {
      const separator = iconsURL.includes("?") ? "&" : "?";
      const response = await fetch(
        `${iconsURL}${separator}q=${encodeURIComponent(query)}&offset=${offset}`,
        { headers: { Accept: "application/json" } },
      );
      if (!response.ok || currentRequest !== requestNumber) return;

      const value: unknown = await response.json();
      if (!isIconPage(value) || currentRequest !== requestNumber) return;

      const options: IconOption[] = [];

      if (
        !append &&
        (!query || "no icon none empty".includes(query.toLocaleLowerCase()))
      ) {
        options.push({ name: "", label: "No icon", svg: "" });
      }

      options.push(...value.items);
      iconGrid.append(...options.map(optionButton));
      nextOffset += value.items.length;
      hasMore = value.has_more;

      const selected = ownerValue();

      for (const option of iconGrid.querySelectorAll<HTMLElement>(
        "[data-icon-name]",
      )) {
        option.classList.toggle(
          "selected",
          option.dataset.iconName === selected,
        );
      }

      emptyState.hidden = iconGrid.childElementCount !== 0;
    } catch {
      if (currentRequest === requestNumber) hasMore = false;
    } finally {
      if (currentRequest === requestNumber) {
        loading = false;
        iconGrid.removeAttribute("aria-busy");
      }
    }
  }

  // Opens picker.
  async function openPicker(owner: HTMLElement): Promise<void> {
    activeOwner = owner;
    searchInput.value = "";
    dialog.showModal();
    await loadOptions("");
    requestAnimationFrame(() => searchInput.focus());
  }

  // Closes picker.
  function closePicker(): void {
    dialog.close();
    activeOwner = null;
  }

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-icon-picker-open]",
  )) {
    button.addEventListener("click", () => {
      const owner = button.closest<HTMLElement>("[data-icon-picker-owner]");
      if (owner) void openPicker(owner);
    });
  }

  searchInput.addEventListener("input", () => {
    if (searchTimer !== undefined) clearTimeout(searchTimer);
    searchTimer = setTimeout(
      () => void loadOptions(searchInput.value.trim()),
      120,
    );
  });
  iconGrid.addEventListener("scroll", () => {
    if (
      iconGrid.scrollHeight - iconGrid.scrollTop - iconGrid.clientHeight <
      160
    ) {
      void loadOptions(currentQuery, { append: true });
    }
  });
  close.addEventListener("click", closePicker);
  dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === dialog) closePicker();
  });
}

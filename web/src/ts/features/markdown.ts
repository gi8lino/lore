// Client-side enhancements for rendered Markdown.

import { copyText } from "../core/clipboard.ts";

type MarkdownRoot = Document | HTMLElement;
type SortDirection = "ascending" | "descending";

type MermaidModule = {
  initialize(options: Record<string, unknown>): void;
  run(options: { nodes: NodeListOf<Element> }): Promise<void>;
};

// Wires markdown tabs behavior.
function setupMarkdownTabs(group: HTMLElement): void {
  const tabs = [
    ...group.querySelectorAll<HTMLButtonElement>(
      ":scope > .markdown-tab-list > .markdown-tab",
    ),
  ];
  const panels = [
    ...group.querySelectorAll<HTMLElement>(
      ":scope > .markdown-tab-panels > .markdown-tab-panel",
    ),
  ];
  if (!tabs.length || tabs.length !== panels.length) return;

  function activate(index: number, focus = false): void {
    for (const [candidateIndex, tab] of tabs.entries()) {
      const active = candidateIndex === index;

      tab.classList.toggle("active", active);
      tab.setAttribute("aria-selected", String(active));
      tab.tabIndex = active ? 0 : -1;
      panels[candidateIndex].classList.toggle(
        "markdown-tab-panel-hidden",
        !active,
      );
    }
    if (focus) tabs[index]?.focus();
  }

  for (const [index, tab] of tabs.entries()) {
    tab.addEventListener("click", () => activate(index));
    tab.addEventListener("keydown", (event: KeyboardEvent) => {
      let next = index;

      switch (event.key) {
        case "ArrowRight":
          next = (index + 1) % tabs.length;
          break;
        case "ArrowLeft":
          next = (index - 1 + tabs.length) % tabs.length;
          break;
        case "Home":
          next = 0;
          break;
        case "End":
          next = tabs.length - 1;
          break;
        default:
          return;
      }

      event.preventDefault();
      activate(next, true);
    });
  }

  const initial = Math.max(
    0,
    tabs.findIndex((tab) => tab.classList.contains("active")),
  );

  activate(initial);
}

const tableCollator = new Intl.Collator(undefined, {
  numeric: true,
  sensitivity: "base",
});

function tableRows(table: HTMLTableElement): HTMLTableRowElement[] {
  return table.tBodies.length ? [...table.tBodies[0].rows] : [];
}

function tableCellText(row: HTMLTableRowElement, column: number): string {
  return (row.cells[column]?.textContent ?? "").trim();
}

function compareTableRows(
  left: HTMLTableRowElement,
  right: HTMLTableRowElement,
  column: number,
  direction: SortDirection,
): number {
  const result = tableCollator.compare(
    tableCellText(left, column),
    tableCellText(right, column),
  );
  return direction === "ascending" ? result : -result;
}

function tableControlIcon(pathData: string): SVGSVGElement {
  const namespace = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(namespace, "svg");

  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");

  const path = document.createElementNS(namespace, "path");

  path.setAttribute("d", pathData);
  svg.append(path);
  return svg;
}

function setupTableSorting(
  table: HTMLTableElement,
  headers: HTMLTableCellElement[],
  body: HTMLTableSectionElement,
): void {
  for (const [column, header] of headers.entries()) {
    const button = header.querySelector<HTMLButtonElement>(
      ":scope .table-sort-button",
    );
    const indicator = header.querySelector<HTMLElement>(
      ":scope .table-sort-indicator",
    );
    if (!button || !indicator) continue;

    button.addEventListener("click", () => {
      const direction: SortDirection =
        header.getAttribute("aria-sort") === "ascending"
          ? "descending"
          : "ascending";

      for (const candidate of headers) {
        candidate.setAttribute("aria-sort", "none");

        const candidateIndicator = candidate.querySelector<HTMLElement>(
          ":scope .table-sort-indicator",
        );

        if (candidateIndicator) candidateIndicator.textContent = "↕";
      }

      header.setAttribute("aria-sort", direction);
      indicator.textContent = direction === "ascending" ? "↑" : "↓";

      const rows = tableRows(table);

      rows.sort((left, right) =>
        compareTableRows(left, right, column, direction),
      );

      for (const row of rows) body.append(row);
    });
  }
}

function setupTableFiltering(
  table: HTMLTableElement,
  shell: HTMLElement,
  headers: HTMLTableCellElement[],
  filterButtons: HTMLButtonElement[],
): void {
  const rows = tableRows(table);
  if (!rows.length || !filterButtons.length) return;

  const filters = new Map<number, string>();
  let activeColumn = -1;
  let activeButton: HTMLButtonElement | null = null;

  const menu = document.createElement("div");

  menu.className = "table-filter-menu";
  menu.hidden = true;

  const menuHeader = document.createElement("div");

  menuHeader.className = "table-filter-menu-header";

  const menuTitle = document.createElement("strong");

  menuTitle.className = "table-filter-menu-title";
  menuHeader.append(menuTitle);

  const field = document.createElement("label");

  field.className = "table-filter-field";

  const label = document.createElement("span");

  label.className = "sr-only";
  label.textContent = "Filter column";

  const searchIcon = document.createElement("span");

  searchIcon.className = "table-filter-search-icon";
  searchIcon.append(
    tableControlIcon("M21 21l-4.35-4.35M19 11a8 8 0 1 1-16 0 8 8 0 0 1 16 0Z"),
  );

  const input = document.createElement("input");

  input.type = "search";
  input.className = "table-filter-input";
  input.placeholder = "Filter values";
  input.autocomplete = "off";

  const clear = document.createElement("button");

  clear.type = "button";
  clear.className = "table-filter-clear";
  clear.setAttribute("aria-label", "Clear column filter");
  clear.hidden = true;
  clear.append(tableControlIcon("M18 6 6 18M6 6l12 12"));
  field.append(label, searchIcon, input, clear);
  menu.append(menuHeader, field);
  shell.append(menu);

  function updateButtonStates(): void {
    for (const [column, button] of filterButtons.entries()) {
      const active = (filters.get(column) || "") !== "";

      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    }
  }

  function applyFilters(): void {
    for (const row of rows) {
      let matches = true;

      for (const [column, query] of filters.entries()) {
        if (!query) continue;
        if (!tableCellText(row, column).toLocaleLowerCase().includes(query)) {
          matches = false;
          break;
        }
      }

      row.hidden = !matches;
    }
    updateButtonStates();
  }

  function closeMenu({
    focusButton = false,
  }: { focusButton?: boolean } = {}): void {
    activeButton?.setAttribute("aria-expanded", "false");
    menu.hidden = true;

    if (focusButton) activeButton?.focus();

    activeColumn = -1;
    activeButton = null;
  }

  function positionMenu(button: HTMLButtonElement): void {
    const shellRect = shell.getBoundingClientRect();
    const buttonRect = button.getBoundingClientRect();
    const menuWidth = menu.offsetWidth;
    const maxLeft = Math.max(0, shell.clientWidth - menuWidth);
    const preferredLeft = buttonRect.right - shellRect.left - menuWidth;

    menu.style.left = `${Math.max(0, Math.min(preferredLeft, maxLeft))}px`;
    menu.style.top = `${buttonRect.bottom - shellRect.top + 4}px`;
  }

  function openMenu(column: number, button: HTMLButtonElement): void {
    if (activeButton && activeButton !== button)
      activeButton.setAttribute("aria-expanded", "false");

    activeColumn = column;
    activeButton = button;

    const columnLabel =
      headers[column]?.dataset.tableColumnLabel || `Column ${column + 1}`;

    menuTitle.textContent = `Filter ${columnLabel}`;
    input.value = filters.get(column) || "";
    clear.hidden = input.value === "";
    menu.hidden = false;
    button.setAttribute("aria-expanded", "true");
    positionMenu(button);
    input.focus();
    input.select();
  }

  for (const [column, button] of filterButtons.entries()) {
    button.addEventListener("click", () => {
      if (!menu.hidden && activeButton === button) closeMenu();
      else openMenu(column, button);
    });
  }

  input.addEventListener("input", () => {
    if (activeColumn < 0) return;

    const query = input.value.trim().toLocaleLowerCase();

    if (query === "") filters.delete(activeColumn);
    else filters.set(activeColumn, query);

    clear.hidden = query === "";
    applyFilters();
  });
  input.addEventListener("keydown", (event: KeyboardEvent) => {
    if (event.key !== "Escape") return;

    event.preventDefault();
    closeMenu({ focusButton: true });
  });
  clear.addEventListener("click", () => {
    if (activeColumn < 0) return;

    input.value = "";
    filters.delete(activeColumn);
    clear.hidden = true;
    applyFilters();
    input.focus();
  });
  menu.addEventListener("focusout", () => {
    setTimeout(() => {
      const focused = document.activeElement;
      if (
        menu.hidden ||
        (focused && menu.contains(focused)) ||
        activeButton === focused
      )
        return;

      closeMenu();
    }, 0);
  });
  shell.addEventListener("pointerdown", (event: PointerEvent) => {
    const target = event.target;
    if (menu.hidden || !(target instanceof Element) || menu.contains(target))
      return;

    const filterButton = target.closest<HTMLButtonElement>(
      ".table-filter-button",
    );
    if (filterButton && filterButtons.includes(filterButton)) return;

    closeMenu();
  });
  window.addEventListener("resize", () => {
    if (!menu.hidden && activeButton) positionMenu(activeButton);
  });
  shell
    .querySelector<HTMLElement>(":scope > .lore-table-scroll")
    ?.addEventListener("scroll", () => {
      if (!menu.hidden && activeButton) positionMenu(activeButton);
    });
  applyFilters();
}

function setupTableHeaders(
  table: HTMLTableElement,
  shell: HTMLElement,
  sortable: boolean,
  filterable: boolean,
): void {
  const headers = [...(table.tHead?.rows[0]?.cells ?? [])];
  const body = table.tBodies[0];
  if (!headers.length || !body) return;

  const filterButtons: HTMLButtonElement[] = [];

  for (const [column, header] of headers.entries()) {
    const columnLabel = header.textContent?.trim() || `Column ${column + 1}`;

    header.dataset.tableColumnLabel = columnLabel;

    if (sortable) header.setAttribute("aria-sort", "none");

    const content = document.createElement("span");

    content.className = "table-header-label";

    while (header.firstChild) content.append(header.firstChild);

    const headerCell = document.createElement("div");

    headerCell.className = "table-header-cell";

    if (sortable) {
      const sortButton = document.createElement("button");

      sortButton.type = "button";
      sortButton.className = "table-sort-button";
      sortButton.setAttribute("aria-label", `Sort by ${columnLabel}`);

      const indicator = document.createElement("span");

      indicator.className = "table-sort-indicator";
      indicator.textContent = "↕";
      indicator.setAttribute("aria-hidden", "true");
      sortButton.append(content, indicator);
      headerCell.append(sortButton);
    } else headerCell.append(content);

    if (filterable) {
      const filterButton = document.createElement("button");

      filterButton.type = "button";
      filterButton.className = "table-filter-button";
      filterButton.setAttribute("aria-label", `Filter ${columnLabel}`);
      filterButton.setAttribute("aria-expanded", "false");
      filterButton.setAttribute("aria-pressed", "false");
      filterButton.title = `Filter ${columnLabel}`;
      filterButton.append(
        tableControlIcon("M22 3H2l8 9.46V19l4 2v-8.54L22 3Z"),
      );
      headerCell.append(filterButton);
      filterButtons.push(filterButton);
    }

    header.append(headerCell);
  }

  if (sortable) setupTableSorting(table, headers, body);
  if (filterable) setupTableFiltering(table, shell, headers, filterButtons);
}

function setupInteractiveTable(table: HTMLTableElement): void {
  if (table.dataset.loreInteractiveReady === "true") return;

  const sortable = table.classList.contains("lore-table-sortable");
  const filterable = table.classList.contains("lore-table-filterable");
  if (!sortable && !filterable) return;

  table.dataset.loreInteractiveReady = "true";

  const shell = document.createElement("div");

  shell.className = "lore-table-shell";

  const scroll = document.createElement("div");

  scroll.className = "lore-table-scroll";
  table.before(shell);
  shell.append(scroll);
  scroll.append(table);
  setupTableHeaders(table, shell, sortable, filterable);
}

// Wires markdown enhancements behavior.
export function setupMarkdownEnhancements(root: MarkdownRoot = document): void {
  for (const group of root.querySelectorAll<HTMLElement>(".markdown-tabs"))
    setupMarkdownTabs(group);
  for (const table of root.querySelectorAll<HTMLTableElement>(
    "table.lore-table-sortable, table.lore-table-filterable",
  ))
    setupInteractiveTable(table);

  setupCodeCopyButtons(root);
}

function setupCodeCopyButtons(root: MarkdownRoot = document): void {
  for (const pre of root.querySelectorAll<HTMLPreElement>(".prose pre")) {
    const code = pre.querySelector<HTMLElement>(":scope > code");
    if (
      !code ||
      code.classList.contains("language-mermaid") ||
      pre.parentElement?.classList.contains("code-block")
    )
      continue;

    const wrapper = document.createElement("div");

    wrapper.className = "code-block";
    pre.before(wrapper);
    wrapper.append(pre);

    const button = document.createElement("button");

    button.type = "button";
    button.className = "code-copy-button";
    button.textContent = "Copy";
    button.setAttribute("aria-label", "Copy code to clipboard");
    wrapper.append(button);

    let resetTimer: ReturnType<typeof setTimeout> | undefined;

    button.addEventListener("click", async () => {
      if (resetTimer) clearTimeout(resetTimer);

      button.disabled = true;

      try {
        await copyText(code.textContent ?? "");
        button.textContent = "Copied";
        button.classList.add("copied");
      } catch (error) {
        console.error("copy code failed", error);
        button.textContent = "Copy failed";
      } finally {
        button.disabled = false;
        resetTimer = setTimeout(() => {
          button.textContent = "Copy";
          button.classList.remove("copied");
        }, 1600);
      }
    });
  }
}

let mermaidModule: MermaidModule | undefined;

// Renders mermaid.
export async function renderMermaid(
  root: MarkdownRoot = document,
  force = false,
): Promise<void> {
  if (!force && document.body?.dataset.renderMermaid !== "true") return;

  const blocks = [
    ...root.querySelectorAll<HTMLElement>("pre code.language-mermaid"),
  ];
  if (!blocks.length) return;

  if (!mermaidModule) {
    const mermaidURL =
      "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
    const imported = (await import(mermaidURL)) as { default: MermaidModule };

    mermaidModule = imported.default;
  }
  for (const block of blocks) {
    const diagram = document.createElement("div");

    diagram.className = "mermaid";
    diagram.textContent = block.textContent;
    block.parentElement?.replaceWith(diagram);
  }

  mermaidModule.initialize({
    startOnLoad: false,
    theme:
      document.documentElement.style.colorScheme === "dark"
        ? "dark"
        : "default",
    securityLevel: "strict",
  });
  await mermaidModule.run({ nodes: root.querySelectorAll(".mermaid") });
}

// Initializes markdown.
export async function initMarkdown(): Promise<void> {
  setupMarkdownEnhancements(document);
  await renderMermaid(document);
}

// Reusable-variable detection and autocomplete for the Markdown editor.

import { requestJSON } from "../../core/http.ts";
import { textareaCaretOffset } from "../../core/textarea.ts";
import { parseEditorCatalog, type CatalogSnippet } from "./catalog.ts";

export interface VariableTrigger {
  start: number;
  query: string;
}

type Fence = { character: string; length: number };

// Reports whether a caret offset is inside fenced code, where Lore macros stay literal.
function fencedCodeAt(value: string, caret: number): boolean {
  const lines = value.slice(0, caret).split("\n");
  let fence: Fence | null = null;

  for (const line of lines) {
    const match = line.match(/^ {0,3}(`{3,}|~{3,})/u);
    if (!match) continue;

    const marker = match[1];
    const next: Fence = { character: marker[0], length: marker.length };
    if (!fence) {
      fence = next;
      continue;
    }

    if (fence.character === next.character && next.length >= fence.length)
      fence = null;
  }

  return fence !== null;
}

// Finds an active {{ variable trigger at the caret.
export function variableTrigger(
  value: string,
  caret: number,
): VariableTrigger | null {
  if (fencedCodeAt(value, caret)) return null;

  const lineStart = value.lastIndexOf("\n", Math.max(0, caret - 1)) + 1;
  const fragment = value.slice(lineStart, caret);
  const opening = fragment.lastIndexOf("{{");
  if (opening < 0) return null;

  const query = fragment.slice(opening + 2);
  if (/[{}]/u.test(query)) return null;

  // Once an author starts an explicit Lore macro, leave it alone. The short
  // {{ trigger is intentionally the variable picker shortcut.
  if (/^(?:var|snippet|include):/iu.test(query)) return null;

  return { start: lineStart + opening, query };
}

// Builds the canonical variable macro for a stored variable name.
export function variableReplacement(name: string): string {
  return `{{var:${name}}}`;
}

function matchingVariables(
  variables: CatalogSnippet[],
  query: string,
): CatalogSnippet[] {
  const normalized = query.trim().toLocaleLowerCase();
  if (!normalized) return variables;

  return variables.filter((variable) =>
    `${variable.name} ${variable.description ?? ""}`
      .toLocaleLowerCase()
      .includes(normalized),
  );
}

// Wires reusable-variable autocomplete behavior.
function setupVariableAutocomplete(source: HTMLTextAreaElement): void {
  const anchor = source.parentElement;
  if (!anchor) return;

  const suggestionAnchor = anchor;
  const menu = document.createElement("div");

  menu.className = "editor-suggestion-menu editor-variable-menu";
  menu.id = "variable-suggestions";
  menu.hidden = true;
  menu.setAttribute("role", "listbox");
  menu.setAttribute("aria-label", "Insert a variable");
  suggestionAnchor.append(menu);

  let trigger: VariableTrigger | null = null;
  let variables: CatalogSnippet[] | null = null;
  let variableLoad: Promise<CatalogSnippet[]> | null = null;
  let results: CatalogSnippet[] = [];
  let active = -1;
  let request = 0;

  function close(): void {
    request += 1;
    trigger = null;
    results = [];
    active = -1;
    menu.hidden = true;
    menu.replaceChildren();

    if (source.getAttribute("aria-controls") === menu.id) {
      source.removeAttribute("aria-controls");
      source.setAttribute("aria-expanded", "false");
    }
  }

  function position(): void {
    if (menu.hidden) return;

    const sourceRect = source.getBoundingClientRect();
    const anchorRect = suggestionAnchor.getBoundingClientRect();
    const caret = textareaCaretOffset(source, source.selectionStart ?? 0);
    const menuWidth = Math.min(
      390,
      Math.max(260, suggestionAnchor.clientWidth - 16),
    );
    const rawLeft = sourceRect.left - anchorRect.left + caret.left;
    const maxLeft = Math.max(8, suggestionAnchor.clientWidth - menuWidth - 8);

    menu.style.width = `${menuWidth}px`;
    menu.style.left = `${Math.max(8, Math.min(rawLeft, maxLeft))}px`;
    menu.style.top = `${sourceRect.top - anchorRect.top + caret.top}px`;
  }

  function render(): void {
    menu.replaceChildren();

    if (!results.length) {
      const empty = document.createElement("div");

      empty.className = "editor-suggestion-empty";
      empty.textContent = trigger?.query
        ? "No matching variables."
        : "No variables available.";
      menu.append(empty);
    } else {
      for (const [index, variable] of results.entries()) {
        const option = document.createElement("button");

        option.type = "button";
        option.className = "editor-suggestion-option";
        option.dataset.variableIndex = String(index);
        option.setAttribute("role", "option");
        option.setAttribute("aria-selected", String(index === active));

        const name = document.createElement("strong");
        const detail = document.createElement("small");

        name.textContent = variable.name;
        detail.textContent =
          variable.description || variableReplacement(variable.name);
        option.append(name, detail);
        menu.append(option);
      }
    }

    menu.hidden = false;
    source.setAttribute("aria-controls", menu.id);
    source.setAttribute("aria-expanded", "true");
    position();
  }

  function choose(index: number): void {
    const variable = results[index];
    const currentTrigger = trigger;
    if (!variable || !currentTrigger) return;

    const end = source.selectionStart ?? currentTrigger.start;

    source.setRangeText(
      variableReplacement(variable.name),
      currentTrigger.start,
      end,
      "end",
    );
    source.dispatchEvent(new Event("input", { bubbles: true }));
    close();
    source.focus();
  }

  async function loadVariables(): Promise<CatalogSnippet[]> {
    if (variables) return variables;
    if (variableLoad) return variableLoad;

    variableLoad = requestJSON("/api/editor/catalog")
      .then((payload) =>
        parseEditorCatalog(payload)
          .snippets.filter((item) => item.kind === "variable")
          .sort((left, right) => left.name.localeCompare(right.name)),
      )
      .then((items) => {
        variables = items;
        return items;
      })
      .finally(() => {
        variableLoad = null;
      });

    return variableLoad;
  }

  async function refresh(): Promise<void> {
    const next = variableTrigger(source.value, source.selectionStart ?? 0);
    if (!next) {
      close();
      return;
    }

    trigger = next;
    const currentRequest = ++request;

    try {
      const available = await loadVariables();
      if (currentRequest !== request) return;

      results = matchingVariables(available, next.query);
      active = results.length ? 0 : -1;
      render();
    } catch (error) {
      console.error("variable catalog failed", error);
      if (currentRequest === request) close();
    }
  }

  source.addEventListener("input", () => void refresh());
  source.addEventListener("click", () => void refresh());
  source.addEventListener("scroll", position);
  source.addEventListener("keydown", (event: KeyboardEvent) => {
    if (menu.hidden) return;

    switch (event.key) {
      case "ArrowDown":
        if (results.length) {
          event.preventDefault();
          active = (active + 1) % results.length;
          render();
        }
        break;
      case "ArrowUp":
        if (results.length) {
          event.preventDefault();
          active = (active - 1 + results.length) % results.length;
          render();
        }
        break;
      case "Enter":
      case "Tab":
        if (active >= 0) {
          event.preventDefault();
          choose(active);
        }
        break;
      case "Escape":
        event.preventDefault();
        close();
        break;
    }
  });

  menu.addEventListener("mousedown", (event: MouseEvent) =>
    event.preventDefault(),
  );
  menu.addEventListener("click", (event: MouseEvent) => {
    const target = event.target;
    if (!(target instanceof Element)) return;

    const option = target.closest<HTMLElement>("[data-variable-index]");
    if (option) choose(Number(option.dataset.variableIndex));
  });

  document.addEventListener("click", (event: MouseEvent) => {
    const target = event.target;
    if (target !== source && target instanceof Node && !menu.contains(target))
      close();
  });
  window.addEventListener("resize", position);
}

// Initializes reusable-variable autocomplete in Markdown editors.
export function initVariableAutocomplete(): void {
  for (const source of document.querySelectorAll<HTMLTextAreaElement>(
    "textarea[data-variable-autocomplete]",
  ))
    setupVariableAutocomplete(source);
}

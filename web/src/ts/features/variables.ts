// Reading-page variable inspection. Saved values are never edited here.

import { requiredElement } from "../core/dom.ts";

export interface VariableField {
  name: string;
  value: string;
  saved: string;
}

// Preserve explicit empty values and whitespace, and omit unchanged fields.
// Object.fromEntries safely handles names such as "__proto__" and "constructor".
export function variableOverrides(
  fields: Iterable<VariableField>,
): Record<string, string> {
  return Object.fromEntries(
    [...fields]
      .filter((field) => field.value !== field.saved)
      .map((field) => [field.name, field.value]),
  );
}

export function initPageVariables(): void {
  const panel = document.querySelector<HTMLElement>("[data-variables-panel]");
  const button = document.querySelector<HTMLButtonElement>(
    "[data-variables-open]",
  );
  const content = document.querySelector<HTMLElement>(
    "[data-variable-content]",
  );
  if (!panel || !button || !content) return;
  setupPageVariables(panel, button, content);
}

function setupPageVariables(
  panel: HTMLElement,
  button: HTMLButtonElement,
  content: HTMLElement,
): void {
  const toggle = requiredElement<HTMLInputElement>(
    panel,
    "[data-variables-highlight]",
  );
  const close = requiredElement<HTMLButtonElement>(
    panel,
    "[data-variables-close]",
  );
  const tooltip = requiredElement<HTMLElement>(
    document,
    "[data-variable-tooltip]",
  );
  const tooltipName = requiredElement<HTMLElement>(
    tooltip,
    "[data-variable-tooltip-name]",
  );
  const tooltipValue = requiredElement<HTMLElement>(
    tooltip,
    "[data-variable-tooltip-value]",
  );
  const values = new Map<string, string>();
  for (const entry of panel.querySelectorAll<HTMLElement>(
    "[data-variable-entry]",
  )) {
    const name = entry.dataset.variableEntry;
    if (name === undefined) continue;
    values.set(
      name,
      requiredElement<HTMLElement>(entry, "[data-variable-saved]")
        .textContent ?? "",
    );
  }
  const marks = [
    ...content.querySelectorAll<HTMLElement>("[data-page-variable]"),
  ].filter((mark) => values.has(mark.dataset.pageVariable ?? ""));
  let active: HTMLElement | null = null;

  function hideTooltip(): void {
    active?.removeAttribute("aria-describedby");
    active = null;
    tooltip.hidden = true;
  }

  function showTooltip(mark: HTMLElement): void {
    if (!toggle.checked) return;
    hideTooltip();
    const name = mark.dataset.pageVariable ?? "";
    tooltipName.textContent = name;
    const saved = values.get(name) ?? "";
    tooltipValue.textContent = saved || "(empty value)";
    tooltip.hidden = false;
    active = mark;
    mark.setAttribute("aria-describedby", tooltip.id);
    const rect = mark.getBoundingClientRect();
    const gap = 8;
    const left = Math.min(
      Math.max(gap, rect.left),
      Math.max(gap, window.innerWidth - tooltip.offsetWidth - gap),
    );
    const below = rect.bottom + gap;
    const top =
      below + tooltip.offsetHeight < window.innerHeight
        ? below
        : Math.max(gap, rect.top - tooltip.offsetHeight - gap);
    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
  }

  function setHighlights(): void {
    content.classList.toggle("variables-highlighted", toggle.checked);
    button.classList.toggle("active", toggle.checked);
    for (const mark of marks) {
      if (toggle.checked) mark.tabIndex = 0;
      else mark.removeAttribute("tabindex");
    }
    hideTooltip();
  }

  function setPanel(open: boolean): void {
    panel.hidden = !open;
    button.setAttribute("aria-expanded", String(open));
    if (open) panel.scrollIntoView({ block: "start" });
  }

  button.addEventListener("click", () => setPanel(Boolean(panel.hidden)));
  close.addEventListener("click", () => {
    setPanel(false);
    button.focus();
  });
  toggle.addEventListener("change", setHighlights);
  for (const mark of marks) {
    mark.addEventListener("pointerenter", (event: PointerEvent) => {
      if (event.pointerType !== "touch") showTooltip(mark);
    });
    mark.addEventListener("pointerleave", () => {
      if (document.activeElement !== mark) hideTooltip();
    });
    mark.addEventListener("focus", () => showTooltip(mark));
    mark.addEventListener("blur", hideTooltip);
    mark.addEventListener("click", (event: MouseEvent) => {
      if (!toggle.checked) return;
      // Inspection mode favors showing provenance over following a surrounding
      // link. Turn highlighting off to use that link normally.
      event.preventDefault();
      showTooltip(mark);
    });
  }
  document.addEventListener("keydown", (event: KeyboardEvent) => {
    if (event.key === "Escape") hideTooltip();
  });
  document.addEventListener("pointerdown", (event: PointerEvent) => {
    if (
      event.target instanceof Node &&
      !active?.contains(event.target) &&
      !tooltip.contains(event.target)
    )
      hideTooltip();
  });
  window.addEventListener("scroll", hideTooltip, true);
  window.addEventListener("resize", hideTooltip);
  setHighlights();
}

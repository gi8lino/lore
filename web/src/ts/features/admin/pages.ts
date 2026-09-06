// Administrator bulk page operations.

import { requestConfirmation } from "../../core/dialogs.ts";

// Wires bulk page selection and destructive-action confirmation.
export function initAdminPages(): void {
  const bulk = document.querySelector<HTMLFormElement>("[data-bulk-pages]");
  if (!bulk) return;

  const pages = [
    ...bulk.querySelectorAll<HTMLInputElement>("[data-bulk-page]"),
  ];
  const selectAll = bulk.querySelector<HTMLInputElement>(
    "[data-bulk-select-all]",
  );
  const count = bulk.querySelector<HTMLElement>("[data-bulk-count]");
  const action = bulk.querySelector<HTMLSelectElement>("[data-bulk-action]");
  const submit = bulk.querySelector<HTMLButtonElement>("[data-bulk-submit]");
  const fields = [...bulk.querySelectorAll<HTMLElement>("[data-bulk-field]")];

  // Refreshes the active administration page state.
  const refresh = (): void => {
    const selected = pages.filter((input) => input.checked).length;

    if (count) count.textContent = `${selected} selected`;
    if (selectAll) {
      selectAll.checked = selected > 0 && selected === pages.length;
      selectAll.indeterminate = selected > 0 && selected < pages.length;
    }
    if (submit) submit.disabled = !selected || !action?.value;

    fields.forEach((field) => {
      field.hidden = field.dataset.bulkField !== action?.value;
    });
  };

  pages.forEach((input) => input.addEventListener("change", refresh));
  selectAll?.addEventListener("change", () => {
    pages.forEach((input) => {
      input.checked = selectAll.checked;
    });
    refresh();
  });
  action?.addEventListener("change", refresh);
  bulk.addEventListener("submit", async (event: SubmitEvent) => {
    if (bulk.dataset.confirming === "true" || action?.value !== "delete") {
      return;
    }

    event.preventDefault();

    const selected = pages.filter((input) => input.checked).length;
    const accepted = await requestConfirmation(
      `Move ${selected} selected page${selected === 1 ? "" : "s"} to the recycle bin?`,
      {
        eyebrow: "Bulk pages",
        title: "Delete selected pages?",
        confirmLabel: "Move to bin",
        cancelLabel: "Cancel",
        danger: true,
      },
    );
    if (!accepted) return;

    bulk.dataset.confirming = "true";
    bulk.requestSubmit();
  });
  refresh();
}

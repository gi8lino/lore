// Administrator feature bootstrap.

import { requestConfirmation } from "../../core/dialogs.ts";
import { errorMessage, responseProblem } from "../../core/http.ts";
import { renderMermaid } from "../markdown.ts";
import { setupGroupMemberPicker } from "./groups.ts";
import { setupNavigationIconPicker } from "./navigation.ts";
import { setupAdminUserEditor, setupPendingOIDCEditor } from "./users.ts";

// Initializes admin.
export function initAdmin(): void {
  const navigationIconPicker = document.querySelector<HTMLDialogElement>(
    "[data-icon-picker-dialog]",
  );
  if (navigationIconPicker) setupNavigationIconPicker(navigationIconPicker);
  for (const card of document.querySelectorAll<HTMLElement>(
    "[data-group-members]",
  ))
    setupGroupMemberPicker(card);

  const userEditor = document.querySelector<HTMLDialogElement>(
    "[data-admin-user-dialog]",
  );
  if (userEditor) setupAdminUserEditor(userEditor);

  const identityEditor = document.querySelector<HTMLDialogElement>(
    "[data-pending-oidc-dialog]",
  );
  if (identityEditor) setupPendingOIDCEditor(identityEditor);

  setupAuthenticationSettings();
  setupPDFSettings();

  const mermaidPreview = document.querySelector<HTMLElement>(
    "[data-rendering-mermaid-preview]",
  );
  if (mermaidPreview) {
    void renderMermaid(mermaidPreview, true).catch((error: unknown) => {
      console.error("rendering Mermaid preview failed", error);
    });
  }

  const bulk = document.querySelector<HTMLFormElement>("[data-bulk-pages]");
  if (bulk) {
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
      if (selectAll)
        selectAll.checked = selected > 0 && selected === pages.length;
      if (selectAll)
        selectAll.indeterminate = selected > 0 && selected < pages.length;
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
      if (bulk.dataset.confirming === "true" || action?.value !== "delete")
        return;
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
}

// Wires the PDF integration test to the endpoint currently entered in the form.
function setupPDFSettings(): void {
  const form = document.querySelector<HTMLFormElement>("[data-pdf-settings]");
  if (!form) return;

  const endpoint = form.elements.namedItem("pdf_url");
  const button = form.querySelector<HTMLButtonElement>("[data-pdf-test]");
  const status = form.querySelector<HTMLElement>("[data-pdf-test-status]");
  if (!(endpoint instanceof HTMLInputElement) || !button || !status) return;

  const setStatus = (
    state: "" | "testing" | "success" | "error",
    message: string,
  ): void => {
    status.dataset.state = state;
    status.textContent = message;
  };

  endpoint.addEventListener("input", () => setStatus("", ""));
  button.addEventListener("click", async () => {
    const pdfURL = endpoint.value.trim();
    if (!pdfURL) {
      setStatus("error", "Enter a PDF service URL to test.");
      endpoint.focus();
      return;
    }

    button.disabled = true;
    setStatus("testing", "Rendering a small test document…");
    try {
      const body = new URLSearchParams({ pdf_url: pdfURL });
      const response = await fetch("/admin/pdf/test", {
        method: "POST",
        body,
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) throw await responseProblem(response);
      const payload = (await response.json()) as { message?: string };
      setStatus("success", payload.message || "PDF service is reachable.");
    } catch (error: unknown) {
      setStatus("error", errorMessage(error));
    } finally {
      button.disabled = false;
    }
  });
}

// Shows only the fields used by the selected browser authentication mode.
function setupAuthenticationSettings(): void {
  const form = document.querySelector<HTMLFormElement>("[data-auth-settings]");
  if (!form) return;

  const mode = form.querySelector<HTMLSelectElement>("[data-auth-mode]");
  const sections = [
    ...form.querySelectorAll<HTMLElement>("[data-auth-fields]"),
  ];
  if (!mode) return;

  const refresh = (): void => {
    sections.forEach((section) => {
      section.hidden = section.dataset.authFields !== mode.value;
    });
  };

  mode.addEventListener("change", refresh);
  refresh();

  const groupSync = form.querySelector<HTMLInputElement>(
    "[data-oidc-group-sync-toggle]",
  );
  const groupOptions = form.querySelector<HTMLElement>(
    "[data-oidc-group-sync-options]",
  );
  const mappingList = form.querySelector<HTMLElement>(
    "[data-oidc-group-mapping-list]",
  );
  const mappingTemplate = form.querySelector<HTMLTemplateElement>(
    "[data-oidc-group-mapping-template]",
  );
  const addMapping = form.querySelector<HTMLButtonElement>(
    "[data-oidc-group-mapping-add]",
  );

  // Removes one mapping row without changing the saved configuration yet.
  const bindMapping = (row: HTMLElement): void => {
    row
      .querySelector<HTMLButtonElement>("[data-oidc-group-mapping-remove]")
      ?.addEventListener("click", () => row.remove());
  };
  mappingList
    ?.querySelectorAll<HTMLElement>("[data-oidc-group-mapping-row]")
    .forEach(bindMapping);

  addMapping?.addEventListener("click", () => {
    const row = mappingTemplate?.content.firstElementChild?.cloneNode(true) as
      HTMLElement | null | undefined;
    if (!row || !mappingList) return;
    mappingList.append(row);
    bindMapping(row);
    row
      .querySelector<HTMLInputElement>('input[name="oidc_group_source"]')
      ?.focus();
  });

  // Keeps optional synchronization details out of the way while disabled.
  const refreshGroupSync = (): void => {
    if (groupOptions) groupOptions.hidden = !groupSync?.checked;
  };
  groupSync?.addEventListener("change", refreshGroupSync);
  refreshGroupSync();
}

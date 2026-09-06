// Administrator authentication, rendering, and PDF configuration behavior.

import { errorMessage, responseProblem } from "../../core/http.ts";
import { renderMermaid } from "../markdown.ts";

// Formats one PDF byte size for the administrator test result.
function formatPDFSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;

  const kibibytes = bytes / 1024;
  if (kibibytes < 1024) {
    return `${kibibytes.toFixed(kibibytes < 10 ? 1 : 0)} KiB`;
  }

  return `${(kibibytes / 1024).toFixed(1)} MiB`;
}

type PDFTestControls = {
  endpoint: HTMLInputElement;
  button: HTMLButtonElement;
  status: HTMLElement;
  dialog: HTMLDialogElement;
  preview: HTMLIFrameElement;
  openPDF: HTMLAnchorElement;
  pages: HTMLElement;
  pagesCheck: HTMLElement;
  pagesMark: HTMLElement;
  size: HTMLElement;
  previewURL: string;
};

function setPDFTestStatus(
  controls: PDFTestControls,
  state: "" | "testing" | "success" | "error",
  message: string,
): void {
  controls.status.dataset.state = state;
  controls.status.textContent = message;
}

function clearPDFTestPreview(controls: PDFTestControls): void {
  controls.preview.removeAttribute("src");
  controls.openPDF.removeAttribute("href");

  if (controls.previewURL) URL.revokeObjectURL(controls.previewURL);
  controls.previewURL = "";
}

function showPDFTestResult(
  controls: PDFTestControls,
  blob: Blob,
  pageCount: number,
  byteSize: number,
): void {
  clearPDFTestPreview(controls);

  controls.previewURL = URL.createObjectURL(blob);
  controls.preview.src = controls.previewURL;
  controls.openPDF.href = controls.previewURL;
  controls.size.textContent = formatPDFSize(byteSize);

  if (pageCount === 2) {
    controls.pagesCheck.dataset.state = "success";
    controls.pagesMark.textContent = "✓";
    controls.pages.textContent = "2 pages";
  } else if (pageCount > 0) {
    controls.pagesCheck.dataset.state = "warning";
    controls.pagesMark.textContent = "!";
    controls.pages.textContent = `${pageCount} page${pageCount === 1 ? "" : "s"} returned; expected 2`;
  } else {
    controls.pagesCheck.dataset.state = "warning";
    controls.pagesMark.textContent = "!";
    controls.pages.textContent = "Page count unavailable";
  }

  controls.dialog.showModal();
}

async function testPDFEndpoint(controls: PDFTestControls): Promise<void> {
  const pdfURL = controls.endpoint.value.trim();
  if (!pdfURL) {
    setPDFTestStatus(controls, "error", "Enter a PDF service URL to test.");
    controls.endpoint.focus();
    return;
  }

  controls.button.disabled = true;
  setPDFTestStatus(
    controls,
    "testing",
    "Rendering the two-page PDF test document…",
  );

  try {
    const body = new URLSearchParams({ pdf_url: pdfURL });
    const response = await fetch("/admin/pdf/test", {
      method: "POST",
      body,
      credentials: "same-origin",
      headers: { Accept: "application/pdf" },
    });
    if (!response.ok) throw await responseProblem(response);

    const blob = await response.blob();
    const pageCount = Number.parseInt(
      response.headers.get("X-Lore-PDF-Pages") ?? "0",
      10,
    );
    const reportedSize = Number.parseInt(
      response.headers.get("X-Lore-PDF-Size") ?? "0",
      10,
    );
    const byteSize = reportedSize > 0 ? reportedSize : blob.size;

    showPDFTestResult(
      controls,
      blob,
      Number.isFinite(pageCount) ? pageCount : 0,
      byteSize,
    );
    setPDFTestStatus(
      controls,
      "success",
      "PDF service is working. Review the generated test document.",
    );
  } catch (error: unknown) {
    setPDFTestStatus(controls, "error", errorMessage(error));
  } finally {
    controls.button.disabled = false;
  }
}

// Wires the PDF integration test to the endpoint currently entered in the form.
function setupPDFSettings(): void {
  const form = document.querySelector<HTMLFormElement>("[data-pdf-settings]");
  if (!form) return;

  const endpoint = form.elements.namedItem("pdf_url");
  const button = form.querySelector<HTMLButtonElement>("[data-pdf-test]");
  const status = form.querySelector<HTMLElement>("[data-pdf-test-status]");
  const dialog = document.querySelector<HTMLDialogElement>(
    "[data-pdf-test-dialog]",
  );
  if (!(endpoint instanceof HTMLInputElement) || !button || !status || !dialog) {
    return;
  }

  const preview = dialog.querySelector<HTMLIFrameElement>(
    "[data-pdf-test-preview]",
  );
  const openPDF = dialog.querySelector<HTMLAnchorElement>(
    "[data-pdf-test-open]",
  );
  const pages = dialog.querySelector<HTMLElement>("[data-pdf-test-pages]");
  const pagesCheck = dialog.querySelector<HTMLElement>(
    "[data-pdf-test-pages-check]",
  );
  const pagesMark = pagesCheck?.querySelector<HTMLElement>(
    ".pdf-test-check-mark",
  );
  const size = dialog.querySelector<HTMLElement>("[data-pdf-test-size]");
  const closeButtons = [
    ...dialog.querySelectorAll<HTMLButtonElement>("[data-pdf-test-close]"),
  ];
  if (!preview || !openPDF || !pages || !pagesCheck || !pagesMark || !size) {
    return;
  }

  const controls: PDFTestControls = {
    endpoint,
    button,
    status,
    dialog,
    preview,
    openPDF,
    pages,
    pagesCheck,
    pagesMark,
    size,
    previewURL: "",
  };

  endpoint.addEventListener("input", () => setPDFTestStatus(controls, "", ""));
  for (const close of closeButtons) {
    close.addEventListener("click", () => dialog.close());
  }
  dialog.addEventListener("close", () => clearPDFTestPreview(controls));
  button.addEventListener("click", () => void testPDFEndpoint(controls));
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
      | HTMLElement
      | null
      | undefined;
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

function setupMermaidPreview(): void {
  const preview = document.querySelector<HTMLElement>(
    "[data-rendering-mermaid-preview]",
  );
  if (!preview) return;

  void renderMermaid(preview, true).catch((error: unknown) => {
    console.error("rendering Mermaid preview failed", error);
  });
}

// Initializes administrator configuration controls present on the current page.
export function initAdminConfiguration(): void {
  setupAuthenticationSettings();
  setupPDFSettings();
  setupMermaidPreview();
}

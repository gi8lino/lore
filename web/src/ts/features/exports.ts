// Page export and share-dialog behavior.

import { copyText } from "../core/clipboard.ts";
import { showNotice } from "../core/dialogs.ts";
import { requiredAttribute, requiredElement } from "../core/dom.ts";
import { errorMessage, responseProblem } from "../core/http.ts";

// Wires admin export behavior.
function setupAdminExport(): void {
  const exportForm =
    document.querySelector<HTMLFormElement>("[data-export-form]");
  const exportAll =
    exportForm?.querySelector<HTMLInputElement>("[data-export-all]");
  const exportPageCheckboxes = [
    ...(exportForm?.querySelectorAll<HTMLInputElement>('input[name="slug"]') ??
      []),
  ];

  exportAll?.addEventListener("change", () => {
    for (const checkbox of exportPageCheckboxes)
      checkbox.checked = exportAll.checked;
  });

  for (const checkbox of exportPageCheckboxes) {
    checkbox.addEventListener("change", () => {
      if (exportAll)
        exportAll.checked =
          exportPageCheckboxes.length > 0 &&
          exportPageCheckboxes.every((item) => item.checked);
    });
  }

  exportForm?.addEventListener("submit", async (event: SubmitEvent) => {
    if (
      event.submitter instanceof HTMLButtonElement &&
      event.submitter.name === "all"
    )
      return;
    if (exportPageCheckboxes.some((checkbox) => checkbox.checked)) return;

    event.preventDefault();
    await showNotice("Select at least one page to export.", {
      title: "Nothing selected",
    });
  });
}

// Starts a browser download for a response blob.
function downloadBlob(blob: Blob, filename: string): void {
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");

  link.href = objectURL;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
}

// Reads a download filename from response headers.
function downloadFilename(response: Response, fallback: string): string {
  const disposition = response.headers.get("Content-Disposition") || "";
  const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (encoded?.[1]) return decodeURIComponent(encoded[1]);

  const plain = disposition.match(/filename="?([^";]+)"?/i);

  return plain?.[1] || fallback;
}

async function downloadPDF(
  dialog: HTMLDialogElement,
  button: HTMLButtonElement,
  progress: HTMLElement,
  pdfURL: string,
): Promise<void> {
  button.disabled = true;

  let progressVisible = false;
  const progressTimer = setTimeout(() => {
    progress.hidden = false;
    progressVisible = true;
  }, 350);

  try {
    const response = await fetch(pdfURL, {
      headers: { Accept: "application/pdf" },
    });
    if (!response.ok) {
      const payload: unknown = await response.json().catch(() => ({}));
      throw await responseProblem(response, payload);
    }

    const blob = await response.blob();

    downloadBlob(blob, downloadFilename(response, "lore-page.pdf"));
    dialog.close();
  } catch (error) {
    console.error("PDF export failed", error);
    dialog.close();
    await showNotice(errorMessage(error) || "PDF could not be generated.", {
      title: "PDF export failed",
    });
  } finally {
    clearTimeout(progressTimer);

    if (progressVisible) progress.hidden = true;

    button.disabled = false;
  }
}

// Wires share dialog behavior.
function setupShareDialog(dialog: HTMLDialogElement): void {
  const open = requiredElement<HTMLButtonElement>(
    document,
    "[data-share-dialog-open]",
  );
  const close = requiredElement<HTMLButtonElement>(
    dialog,
    "[data-share-dialog-close]",
  );
  const permalink = requiredElement<HTMLButtonElement>(
    dialog,
    "[data-share-permalink]",
  );
  const permalinkStatus = requiredElement<HTMLElement>(
    dialog,
    "[data-share-permalink-status]",
  );
  const print = requiredElement<HTMLButtonElement>(
    dialog,
    "[data-share-print]",
  );
  const pdf = requiredElement<HTMLButtonElement>(dialog, "[data-share-pdf]");
  const markdown = requiredElement<HTMLAnchorElement>(
    dialog,
    "[data-share-markdown]",
  );
  const progress = requiredElement<HTMLElement>(
    dialog,
    "[data-share-progress]",
  );
  const permalinkPath = requiredAttribute(permalink, "data-url");
  const pdfURL = requiredAttribute(pdf, "data-url");

  open.addEventListener("click", () => dialog.showModal());
  close.addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === dialog) dialog.close();
  });
  permalink.addEventListener("click", async () => {
    const original = permalinkStatus.textContent;
    try {
      const url = new URL(permalinkPath, window.location.href).href;

      await copyText(url);
      permalinkStatus.textContent = "Copied. Authentication is still required.";
      setTimeout(() => {
        permalinkStatus.textContent = original;
      }, 1800);
    } catch (error) {
      console.error("Permalink copy failed", error);
      dialog.close();
      await showNotice(
        errorMessage(error) || "Permalink could not be copied.",
        {
          title: "Copy failed",
        },
      );
    }
  });

  print.addEventListener("click", () => {
    dialog.close();
    requestAnimationFrame(() => window.print());
  });
  markdown.addEventListener("click", () => dialog.close());

  pdf.addEventListener(
    "click",
    () => void downloadPDF(dialog, pdf, progress, pdfURL),
  );
}

// Initializes exports.
export function initExports(): void {
  setupAdminExport();

  const shareDialog = document.querySelector<HTMLDialogElement>(
    "[data-share-dialog]",
  );

  if (shareDialog) setupShareDialog(shareDialog);
}

// Editor attachment upload and insertion.

import { requestConfirmation } from "../../core/dialogs.ts";
import { insertMarkdownAtSelection } from "./toolbar.ts";

export interface AttachmentItem {
  id: number;
  filename: string;
  size_bytes: number;
  usage_count?: number;
  url: string;
}

function isAttachmentItem(value: unknown): value is AttachmentItem {
  if (typeof value !== "object" || value === null) return false;

  const item = value as Partial<AttachmentItem>;

  return (
    typeof item.id === "number" &&
    typeof item.filename === "string" &&
    typeof item.size_bytes === "number" &&
    typeof item.url === "string"
  );
}

// Builds the Markdown link for an attachment.
export function attachmentMarkdown(
  item: Pick<AttachmentItem, "filename" | "url">,
): string {
  return `[${item.filename}](${item.url})`;
}

// Wires attachment dialog behavior.
function setupAttachmentDialog(form: HTMLFormElement): void {
  const dialog = document.querySelector<HTMLDialogElement>(
    "[data-attachment-dialog]",
  );
  const open = form.querySelector<HTMLButtonElement>(
    "[data-attachment-dialog-open]",
  );
  const textarea = form.querySelector<HTMLTextAreaElement>(
    "[data-markdown-editor]",
  );
  if (!dialog || !open || !textarea) return;

  const close = dialog.querySelector<HTMLButtonElement>(
    "[data-attachment-dialog-close]",
  );
  const body = dialog.querySelector<HTMLElement>(
    "[data-attachment-dialog-body]",
  );
  const upload = dialog.querySelector<HTMLInputElement>(
    "[data-attachment-upload]",
  );
  const status = dialog.querySelector<HTMLElement>("[data-attachment-status]");
  const listURL = dialog.dataset.attachmentListUrl;
  const uploadURL = dialog.dataset.attachmentUploadUrl;
  if (!body || !status || !listURL || !uploadURL) return;

  const attachmentDialog = dialog;
  const attachmentListURL = listURL;
  const attachmentUploadURL = uploadURL;
  const attachmentBody = body;
  const attachmentStatus = status;
  const editor = textarea;

  // Builds one attachment-library row.
  function row(item: AttachmentItem): HTMLElement {
    const element = document.createElement("div");

    element.className = "attachment-row";

    const info = document.createElement("span");
    const name = document.createElement("strong");
    const meta = document.createElement("small");

    name.textContent = item.filename;
    meta.textContent = `${Math.max(1, Math.round(item.size_bytes / 1024))} KiB · ${item.usage_count || 0} reference${item.usage_count === 1 ? "" : "s"}`;
    info.append(name, meta);

    const actions = document.createElement("span");

    actions.className = "attachment-row-actions";

    const insert = document.createElement("button");

    insert.type = "button";
    insert.className = "button";
    insert.textContent = "Insert";
    insert.addEventListener("click", () => {
      insertMarkdownAtSelection(editor, attachmentMarkdown(item));
      attachmentDialog.close();
    });

    const remove = document.createElement("button");

    remove.type = "button";
    remove.className = "button danger";
    remove.textContent = "Delete";
    remove.addEventListener("click", async () => {
      if (
        !(await requestConfirmation(
          `Delete ${item.filename}? Existing Markdown references will stop working.`,
          { title: "Delete attachment", confirmLabel: "Delete", danger: true },
        ))
      )
        return;

      const response = await fetch(`/api/attachments/${item.id}`, {
        method: "DELETE",
      });

      if (response.ok) element.remove();
    });
    actions.append(insert, remove);
    element.append(info, actions);
    return element;
  }

  // Loads attachments into the library dialog.
  async function load(): Promise<void> {
    attachmentBody.innerHTML = '<p class="muted">Loading files…</p>';

    const response = await fetch(attachmentListURL, {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      attachmentBody.innerHTML =
        '<p class="muted">Files could not be loaded.</p>';
      return;
    }

    const payload: unknown = await response.json();
    const items = Array.isArray(payload)
      ? payload.filter(isAttachmentItem)
      : [];

    attachmentBody.replaceChildren();
    if (!items.length) {
      const empty = document.createElement("p");

      empty.className = "muted";
      empty.textContent = "No attachments yet.";
      attachmentBody.append(empty);
      return;
    }

    items.forEach((item) => attachmentBody.append(row(item)));
  }

  open.addEventListener("click", () => {
    attachmentDialog.showModal();
    void load();
  });
  close?.addEventListener("click", () => attachmentDialog.close());
  attachmentDialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === attachmentDialog) attachmentDialog.close();
  });
  upload?.addEventListener("change", async () => {
    const file = upload.files?.[0];
    if (!file) return;

    const data = new FormData();

    data.append("file", file);
    attachmentStatus.textContent = `Uploading ${file.name}…`;

    const response = await fetch(attachmentUploadURL, {
      method: "POST",
      body: data,
    });
    if (!response.ok) {
      attachmentStatus.textContent = "Upload failed.";
      return;
    }

    const payload: unknown = await response.json();
    if (!isAttachmentItem(payload)) {
      attachmentStatus.textContent = "Upload returned an invalid response.";
      return;
    }

    insertMarkdownAtSelection(editor, attachmentMarkdown(payload));
    upload.value = "";
    attachmentStatus.textContent = "Uploaded and inserted.";
    attachmentDialog.close();
  });
}

// Initializes attachments.
export function initAttachments(): void {
  for (const form of document.querySelectorAll<HTMLFormElement>(
    "[data-editor-form]",
  ))
    setupAttachmentDialog(form);
}

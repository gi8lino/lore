// Image upload, library, and Markdown insertion.

import { requestConfirmation, showNotice } from "../core/dialogs.ts";
import { isRecord } from "../core/guards.ts";
import { errorMessage, requestJSON } from "../core/http.ts";
import { insertMarkdownAtSelection } from "./editor/toolbar.ts";

export type ImageItem = {
  id: number;
  filename: string;
  content_type: string;
  size_bytes: number;
  uploaded_by: number;
  uploader: string;
  created_at: string;
  usage_count: number;
  url: string;
};

function isImageItem(value: unknown): value is ImageItem {
  return (
    isRecord(value) &&
    typeof value.id === "number" &&
    typeof value.filename === "string" &&
    typeof value.size_bytes === "number" &&
    typeof value.usage_count === "number" &&
    typeof value.url === "string"
  );
}

// Builds Markdown for an uploaded image.
export function imageMarkdown(
  image: Pick<ImageItem, "filename" | "url">,
): string {
  const label =
    image.filename
      .replace(/\.[^.]+$/, "")
      .replaceAll("[", "")
      .replaceAll("]", "") || "image";
  return `![${label}](${image.url})`;
}

// Uploads one image and returns its stored metadata.
async function uploadImage(url: string, file: File): Promise<ImageItem> {
  const data = new FormData();

  data.append("file", file);

  const payload = await requestJSON(url, {
    method: "POST",
    body: data,
  });
  if (!isImageItem(payload)) throw new Error("Invalid image response.");

  return payload;
}

// Returns image files from a file list.
function imageFiles(files: FileList | readonly File[]): File[] {
  return [...files].filter((file) => file.type.startsWith("image/"));
}

function hasDraggedImageItem(dataTransfer: DataTransfer | null): boolean {
  if (!dataTransfer) return false;

  return [...dataTransfer.items].some((item) => item.type.startsWith("image/"));
}

function hasDraggedImage(dataTransfer: DataTransfer | null): boolean {
  if (!dataTransfer) return false;
  if (imageFiles(dataTransfer.files).length > 0) return true;

  return hasDraggedImageItem(dataTransfer);
}

// Wires media dialog behavior.
function setupMediaDialog(dialog: HTMLDialogElement): void {
  const open = document.querySelector<HTMLElement>("[data-media-dialog-open]");
  const close = dialog.querySelector<HTMLButtonElement>(
    "[data-media-dialog-close]",
  );
  const body = dialog.querySelector<HTMLElement>("[data-media-dialog-body]");
  const uploadInput = dialog.querySelector<HTMLInputElement>(
    "[data-media-upload-input]",
  );
  const uploadStatus = dialog.querySelector<HTMLElement>(
    "[data-media-upload-status]",
  );
  const textarea = document.querySelector<HTMLTextAreaElement>(
    "[data-markdown-editor]",
  );
  const sourcePane =
    textarea?.closest<HTMLElement>("[data-editor-source-pane]") ?? null;
  const dropStatus =
    sourcePane?.querySelector<HTMLElement>("[data-editor-upload-status]") ??
    null;
  if (!open || !close || !body || !uploadInput || !uploadStatus || !textarea)
    return;

  const mediaBody = body;
  const mediaUploadInput = uploadInput;
  const mediaUploadStatus = uploadStatus;
  const editor = textarea;
  const listURL = dialog.dataset.mediaListUrl;
  const uploadURL = dialog.dataset.mediaUploadUrl;
  if (!listURL || !uploadURL) return;

  const mediaListURL = listURL;
  const mediaUploadURL = uploadURL;

  let images: ImageItem[] = [];
  let loaded = false;
  let uploading = false;

  function renderImages(): void {
    mediaBody.replaceChildren();
    if (!images.length) {
      const empty = document.createElement("p");

      empty.className = "muted";
      empty.textContent = "No images uploaded yet.";
      mediaBody.append(empty);
      return;
    }

    for (const image of images) {
      const item = document.createElement("article");

      item.className = "media-dialog-item";

      const preview = document.createElement("img");

      preview.src = image.url;
      preview.alt = "";
      preview.loading = "lazy";

      const info = document.createElement("div");

      info.className = "media-dialog-info";

      const name = document.createElement("strong");

      name.textContent = image.filename;

      const meta = document.createElement("small");
      const kib = image.size_bytes / 1024;
      const size =
        kib >= 1024
          ? `${(kib / 1024).toFixed(1)} MiB`
          : `${kib.toFixed(1)} KiB`;
      const references = `reference${image.usage_count === 1 ? "" : "s"}`;

      meta.textContent = `${size} · ${image.usage_count} ${references}`;
      info.append(name, meta);

      const insert = document.createElement("button");

      insert.type = "button";
      insert.className = "button";
      insert.textContent = "Insert";
      insert.addEventListener("click", () => {
        insertMarkdownAtSelection(editor, imageMarkdown(image));
        dialog.close();
      });

      item.append(preview, info, insert);
      mediaBody.append(item);
    }
  }

  async function loadImages(force = false): Promise<void> {
    if (loaded && !force) return;

    mediaBody.innerHTML = '<p class="muted">Loading images…</p>';

    try {
      const payload = await requestJSON(mediaListURL);

      images = Array.isArray(payload) ? payload.filter(isImageItem) : [];
      loaded = true;
      renderImages();
    } catch (error) {
      console.error("image library failed", error);
      mediaBody.innerHTML =
        '<p class="revision-dialog-error">Images could not be loaded.</p>';
    }
  }

  async function uploadFiles(
    files: FileList | readonly File[],
    { closeDialog = false }: { closeDialog?: boolean } = {},
  ): Promise<void> {
    const accepted = imageFiles(files);
    if (!accepted.length || uploading) return;

    uploading = true;
    mediaUploadInput.disabled = true;
    sourcePane?.classList.add("is-uploading-image");

    try {
      for (let index = 0; index < accepted.length; index += 1) {
        const file = accepted[index];
        const label = `Uploading image ${index + 1} of ${accepted.length}…`;

        mediaUploadStatus.textContent = label;

        if (dropStatus) dropStatus.textContent = label;

        const image = await uploadImage(mediaUploadURL, file);

        images.unshift(image);
        loaded = true;
        insertMarkdownAtSelection(editor, imageMarkdown(image));
      }

      renderImages();

      if (dropStatus) dropStatus.textContent = "Image uploaded and inserted.";
      if (closeDialog && dialog.open) dialog.close();
    } catch (error) {
      console.error("image upload failed", error);

      const message = errorMessage(error) || "Upload failed.";

      mediaUploadStatus.textContent = message;

      if (dropStatus) dropStatus.textContent = message;

      return;
    } finally {
      uploading = false;
      mediaUploadInput.disabled = false;
      mediaUploadInput.value = "";
      sourcePane?.classList.remove("is-uploading-image");
      sourcePane?.classList.remove("is-dragging-image");
    }

    mediaUploadStatus.textContent = "JPEG, PNG, GIF or WebP · max 10 MiB";

    if (dropStatus) {
      setTimeout(() => {
        dropStatus.textContent =
          "Paste or drop images directly into the editor.";
      }, 1800);
    }
  }

  open.addEventListener("click", () => {
    dialog.showModal();
    void loadImages();
  });
  close.addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", (event) => {
    if (event.target === dialog) dialog.close();
  });

  mediaUploadInput.addEventListener("change", () => {
    if (mediaUploadInput.files)
      void uploadFiles(mediaUploadInput.files, { closeDialog: true });
  });

  editor.addEventListener("paste", (event: ClipboardEvent) => {
    const files = imageFiles(event.clipboardData?.files || []);
    if (!files.length) return;

    event.preventDefault();
    void uploadFiles(files);
  });

  if (sourcePane) {
    sourcePane.addEventListener("dragenter", (event: DragEvent) => {
      if (!hasDraggedImage(event.dataTransfer)) return;

      event.preventDefault();
      sourcePane.classList.add("is-dragging-image");
    });
    sourcePane.addEventListener("dragover", (event: DragEvent) => {
      if (!hasDraggedImageItem(event.dataTransfer)) return;

      event.preventDefault();

      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";

      sourcePane.classList.add("is-dragging-image");
    });
    sourcePane.addEventListener("dragleave", (event: DragEvent) => {
      const related = event.relatedTarget;
      if (!(related instanceof Node) || !sourcePane.contains(related))
        sourcePane.classList.remove("is-dragging-image");
    });
    sourcePane.addEventListener("drop", (event: DragEvent) => {
      const files = imageFiles(event.dataTransfer?.files || []);

      sourcePane.classList.remove("is-dragging-image");
      if (!files.length) return;

      event.preventDefault();
      editor.focus();
      void uploadFiles(files);
    });
  }
}

async function deleteMediaImage(button: HTMLButtonElement): Promise<void> {
  const item = button.closest<HTMLElement>("[data-media-settings-item]");
  if (!item) return;

  const usage = Number(button.dataset.mediaUsage || 0);
  const message =
    usage > 0
      ? `Delete this image permanently? It is still referenced ${usage} time${usage === 1 ? "" : "s"} and those references will break.`
      : "Delete this unused image permanently?";
  if (
    !(await requestConfirmation(message, {
      title: "Delete image",
      confirmLabel: "Delete image",
    }))
  )
    return;

  const deleteURL = button.dataset.deleteUrl;
  if (!deleteURL) return;

  button.disabled = true;

  try {
    await requestJSON(deleteURL, { method: "DELETE" });

    item.remove();

    const list = document.querySelector<HTMLElement>(
      "[data-media-settings-list]",
    );
    if (list && !list.querySelector("[data-media-settings-item]")) {
      const empty = document.createElement("p");

      empty.className = "muted";
      empty.dataset.mediaSettingsEmpty = "";
      empty.textContent =
        "No images uploaded yet. Upload one from the page editor.";
      list.append(empty);
    }
  } catch (error) {
    console.error("image deletion failed", error);
    await showNotice(errorMessage(error) || "Image could not be deleted.", {
      title: "Image deletion failed",
    });
    button.disabled = false;
  }
}

// Initializes media.
export function initMedia(): void {
  const mediaDialog = document.querySelector<HTMLDialogElement>(
    "[data-media-dialog]",
  );

  if (mediaDialog) setupMediaDialog(mediaDialog);

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-media-delete]",
  )) {
    button.addEventListener("click", () => void deleteMediaImage(button));
  }
}

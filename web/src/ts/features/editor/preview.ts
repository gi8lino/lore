// Editor write, split, and preview modes.

import { createLatestRequest, isAbortError } from "../../core/async.ts";
import { isRecord } from "../../core/guards.ts";
import { errorMessage, requestJSON } from "../../core/http.ts";
import { renderMermaid, setupMarkdownEnhancements } from "../markdown.ts";
import { preferredEditorMode, rememberEditorMode } from "./experience.ts";

export type EditorMode = "write" | "split" | "preview";

const modes = new Set<EditorMode>(["write", "split", "preview"]);

interface PreviewPayload {
  html: string;
}

function isPreviewPayload(value: unknown): value is PreviewPayload {
  return isRecord(value) && typeof value.html === "string";
}

function editorMode(value: string | undefined): EditorMode {
  return value && modes.has(value as EditorMode)
    ? (value as EditorMode)
    : "write";
}

// Returns labels for an editor workspace mode.
export function editorModeCopy(mode: string): {
  title: string;
  description: string;
} {
  switch (mode) {
    case "split":
      return {
        title: "Markdown & preview",
        description: "Edit Markdown with a live rendered preview.",
      };
    case "preview":
      return {
        title: "Preview",
        description: "Rendered page preview.",
      };
    default:
      return {
        title: "Markdown",
        description: "Markdown stays the source of truth.",
      };
  }
}

// Wires editor preview behavior.
function setupEditorPreview(form: HTMLFormElement): void {
  const workspace = form.querySelector<HTMLElement>("[data-editor-workspace]");
  const source = form.querySelector<HTMLTextAreaElement>(
    "[data-markdown-editor]",
  );
  const slug = form.querySelector<HTMLInputElement>('[name="slug"]');
  const preview = form.querySelector<HTMLElement>("[data-editor-preview]");
  const content = form.querySelector<HTMLElement>(
    "[data-editor-preview-content]",
  );
  const status = form.querySelector<HTMLElement>(
    "[data-editor-preview-status]",
  );
  const sectionTitle = form.querySelector<HTMLElement>(
    "[data-editor-section-title]",
  );
  const sectionDescription = form.querySelector<HTMLElement>(
    "[data-editor-section-description]",
  );
  const buttons = [
    ...form.querySelectorAll<HTMLButtonElement>("[data-editor-mode]"),
  ];
  const previewURL = form.dataset.previewUrl;
  if (
    !workspace ||
    !source ||
    !preview ||
    !content ||
    !status ||
    !buttons.length ||
    !previewURL
  )
    return;

  const previewEndpoint = previewURL;
  const editorWorkspace = workspace;
  const sourceEditor = source;
  const previewPanel = preview;
  const previewContent = content;
  const previewStatus = status;
  const previewRequests = createLatestRequest();
  let timer: ReturnType<typeof setTimeout> | undefined;
  let mode: EditorMode = "write";
  let syncing = false;

  // Renders preview.
  async function renderPreview(): Promise<void> {
    const signal = previewRequests.next();

    previewStatus.hidden = false;
    previewStatus.textContent = "Rendering preview…";

    try {
      const payload = await requestJSON(previewEndpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          markdown: sourceEditor.value,
          slug: slug?.value || "",
        }),
        signal,
      });
      if (!isPreviewPayload(payload))
        throw new Error("Invalid preview response.");

      previewContent.innerHTML = payload.html;
      previewStatus.hidden = true;
      setupMarkdownEnhancements(previewPanel);
      await renderMermaid(previewPanel);
    } catch (error) {
      if (isAbortError(error)) return;

      console.error("markdown preview failed", error);
      previewStatus.hidden = false;
      previewStatus.textContent =
        errorMessage(error) || "Preview could not be rendered.";
    }
  }

  // Schedules preview.
  function schedulePreview(): void {
    if (mode === "write") return;

    if (timer !== undefined) clearTimeout(timer);

    timer = setTimeout(() => void renderPreview(), 180);
  }

  // Sets mode.
  function setMode(nextMode: string | undefined, remember = true): void {
    mode = editorMode(nextMode);
    form.dataset.editorMode = mode;
    editorWorkspace.dataset.editorMode = mode;
    previewPanel.hidden = mode === "write";

    const copy = editorModeCopy(mode);

    if (sectionTitle) sectionTitle.textContent = copy.title;
    if (sectionDescription) sectionDescription.textContent = copy.description;
    for (const button of buttons) {
      const active = button.dataset.editorMode === mode;

      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    }
    if (remember) rememberEditorMode(mode);
    if (mode !== "write") void renderPreview();
  }

  // Synchronizes scroll.
  function syncScroll(from: HTMLElement, to: HTMLElement): void {
    if (mode !== "split" || syncing) return;

    const fromRange = from.scrollHeight - from.clientHeight;
    const toRange = to.scrollHeight - to.clientHeight;
    if (fromRange <= 0 || toRange <= 0) return;

    syncing = true;
    to.scrollTop = (from.scrollTop / fromRange) * toRange;
    requestAnimationFrame(() => (syncing = false));
  }

  for (const button of buttons) {
    button.addEventListener("click", () => setMode(button.dataset.editorMode));
  }

  form.addEventListener("editor:toggle-preview", () =>
    setMode(mode === "write" ? "split" : "write"),
  );
  sourceEditor.addEventListener("input", schedulePreview);
  slug?.addEventListener("input", schedulePreview);
  sourceEditor.addEventListener("scroll", () =>
    syncScroll(sourceEditor, previewPanel),
  );
  previewPanel.addEventListener("scroll", () =>
    syncScroll(previewPanel, sourceEditor),
  );

  setMode(preferredEditorMode(), false);
}

// Initializes editor preview.
export function initEditorPreview(): void {
  for (const form of document.querySelectorAll<HTMLFormElement>(
    "[data-editor-form]",
  ))
    setupEditorPreview(form);
}

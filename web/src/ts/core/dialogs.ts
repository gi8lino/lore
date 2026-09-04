// Reusable confirmation and notice dialog helpers.

interface ConfirmationOptions {
  eyebrow?: string;
  title?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

interface NoticeOptions {
  title?: string;
}

function confirmationDialog(): HTMLDialogElement | null {
  if (typeof document === "undefined") return null;
  return document.querySelector<HTMLDialogElement>("[data-confirm-dialog]");
}

// Returns the shared notice dialog elements.
function noticeDialog(): HTMLDialogElement | null {
  if (typeof document === "undefined") return null;
  return document.querySelector<HTMLDialogElement>("[data-notice-dialog]");
}

// Shows a confirmation dialog and resolves with the user choice.
export function requestConfirmation(
  message: string,
  options: ConfirmationOptions = {},
): Promise<boolean> {
  const dialog = confirmationDialog();
  if (!dialog) return Promise.resolve(false);

  const eyebrow = dialog.querySelector<HTMLElement>(
    "[data-confirm-dialog-eyebrow]",
  );
  const title = dialog.querySelector<HTMLElement>(
    "[data-confirm-dialog-title]",
  );
  const body = dialog.querySelector<HTMLElement>(
    "[data-confirm-dialog-message]",
  );
  const accept = dialog.querySelector<HTMLButtonElement>(
    "[data-confirm-dialog-accept]",
  );
  const cancelLabel = dialog.querySelector<HTMLElement>(
    "[data-confirm-dialog-cancel-label]",
  );
  const cancelButtons = [
    ...dialog.querySelectorAll<HTMLButtonElement>(
      "[data-confirm-dialog-cancel]",
    ),
  ];
  if (!title || !body || !accept) return Promise.resolve(false);

  if (eyebrow) eyebrow.textContent = options.eyebrow || "Confirmation";
  title.textContent = options.title || "Confirm action";
  body.textContent = message || "Continue?";
  accept.textContent = options.confirmLabel || "Continue";
  if (cancelLabel) cancelLabel.textContent = options.cancelLabel || "Cancel";
  accept.classList.toggle("danger", options.danger !== false);
  accept.classList.toggle("primary", options.danger === false);

  return new Promise<boolean>((resolve) => {
    let settled = false;
    // Completes the active dialog request.
    const finish = (value: boolean) => {
      if (settled) return;
      settled = true;
      dialog.close();
      resolve(value);
    };

    for (const button of cancelButtons) button.onclick = () => finish(false);
    accept.onclick = () => finish(true);
    dialog.oncancel = (event: Event) => {
      event.preventDefault();
      finish(false);
    };
    dialog.onclick = (event: MouseEvent) => {
      if (event.target === dialog) finish(false);
    };
    dialog.showModal();
  });
}

// Shows a simple notice dialog.
export function showNotice(
  message: string,
  options: NoticeOptions = {},
): Promise<void> {
  const dialog = noticeDialog();
  if (!dialog) return Promise.resolve();

  const title = dialog.querySelector<HTMLElement>("[data-notice-dialog-title]");
  const body = dialog.querySelector<HTMLElement>(
    "[data-notice-dialog-message]",
  );
  const closeButtons = [
    ...dialog.querySelectorAll<HTMLButtonElement>("[data-notice-dialog-close]"),
  ];
  if (!title || !body) return Promise.resolve();

  title.textContent = options.title || "Notice";
  body.textContent = message || "";

  return new Promise<void>((resolve) => {
    let settled = false;
    // Completes the active dialog request.
    const finish = () => {
      if (settled) return;
      settled = true;
      dialog.close();
      resolve();
    };

    for (const button of closeButtons) button.onclick = finish;
    dialog.oncancel = (event: Event) => {
      event.preventDefault();
      finish();
    };
    dialog.onclick = (event: MouseEvent) => {
      if (event.target === dialog) finish();
    };
    dialog.showModal();
  });
}

// Initializes confirm forms.
export function initConfirmForms(): void {
  if (typeof document === "undefined") return;
  for (const form of document.querySelectorAll<HTMLFormElement>(
    "form[data-confirm]",
  )) {
    form.addEventListener("submit", async (event: SubmitEvent) => {
      if (form.dataset.confirmBypass === "true") {
        delete form.dataset.confirmBypass;
        return;
      }

      event.preventDefault();
      const accepted = await requestConfirmation(
        form.dataset.confirm || "Continue?",
        {
          title: form.dataset.confirmTitle || "Confirm action",
          confirmLabel: form.dataset.confirmLabel || "Continue",
          cancelLabel: form.dataset.confirmCancelLabel || "Cancel",
          eyebrow: form.dataset.confirmEyebrow || "Confirmation",
          danger: form.dataset.confirmDanger !== "false",
        },
      );
      if (!accepted) return;

      form.dataset.confirmBypass = "true";
      if (event.submitter instanceof HTMLElement)
        form.requestSubmit(event.submitter);
      else form.requestSubmit();
    });
  }
}

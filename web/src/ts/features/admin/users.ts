// Administrator user access and external identity editors.

// Returns group identifiers encoded on a user row action.
function selectedGroupIDs(button: HTMLElement): Set<string> {
  return new Set(
    (button.dataset.groupIds || "").trim().split(/\s+/).filter(Boolean),
  );
}

// Wires role and group editing for registered users.
export function setupAdminUserEditor(dialog: HTMLDialogElement): void {
  const form = dialog.querySelector<HTMLFormElement>("[data-admin-user-form]");
  const name = dialog.querySelector<HTMLElement>("[data-admin-user-name]");
  const identity = dialog.querySelector<HTMLElement>(
    "[data-admin-user-identity]",
  );
  const role = dialog.querySelector<HTMLSelectElement>(
    "[data-admin-user-role]",
  );
  const localLogin = dialog.querySelector<HTMLElement>(
    "[data-admin-user-local-login]",
  );
  const localCredentialEnabled = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-local-credential-enabled]",
  );
  const groupInputs = [
    ...dialog.querySelectorAll<HTMLInputElement>('input[name="group_id"]'),
  ];
  if (!form || !name || !identity || !role) return;

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-admin-user-edit]",
  )) {
    button.addEventListener("click", () => {
      const userID = button.dataset.userId;
      if (!userID) return;

      const groups = selectedGroupIDs(button);
      form.action = `/admin/users/${encodeURIComponent(userID)}`;
      name.textContent =
        button.dataset.userName || button.dataset.userUsername || "Edit user";

      const username = button.dataset.userUsername || "";
      const email = button.dataset.userEmail || "";
      identity.textContent = email ? `${username} · ${email}` : username;
      role.value = button.dataset.userRole || "viewer";
      if (localLogin && localCredentialEnabled) {
        const hasLocalCredential =
          button.dataset.userHasLocalCredential === "true";
        localLogin.hidden = !hasLocalCredential;
        localCredentialEnabled.checked =
          button.dataset.userLocalCredentialEnabled === "true";
        localCredentialEnabled.disabled = !hasLocalCredential;
      }
      groupInputs.forEach((input) => {
        input.checked = groups.has(input.value);
      });

      dialog.showModal();
      requestAnimationFrame(() => role.focus());
    });
  }

  for (const button of dialog.querySelectorAll<HTMLButtonElement>(
    "[data-admin-user-close]",
  )) {
    button.addEventListener("click", () => dialog.close());
  }
  dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === dialog) dialog.close();
  });
}

// Wires administrator decisions for unknown verified OIDC identities.
export function setupPendingOIDCEditor(dialog: HTMLDialogElement): void {
  const name = dialog.querySelector<HTMLElement>("[data-pending-oidc-name]");
  const profile = dialog.querySelector<HTMLElement>(
    "[data-pending-oidc-profile]",
  );
  const issuer = dialog.querySelector<HTMLElement>(
    "[data-pending-oidc-issuer]",
  );
  const subject = dialog.querySelector<HTMLElement>(
    "[data-pending-oidc-subject]",
  );
  const user = dialog.querySelector<HTMLSelectElement>(
    "[data-pending-oidc-user]",
  );
  const match = dialog.querySelector<HTMLElement>("[data-pending-oidc-match]");
  const linkForm = dialog.querySelector<HTMLFormElement>(
    "[data-pending-oidc-link-form]",
  );
  const approveForm = dialog.querySelector<HTMLFormElement>(
    "[data-pending-oidc-approve-form]",
  );
  const rejectForm = dialog.querySelector<HTMLFormElement>(
    "[data-pending-oidc-reject-form]",
  );
  if (
    !name ||
    !profile ||
    !issuer ||
    !subject ||
    !user ||
    !linkForm ||
    !approveForm ||
    !rejectForm
  )
    return;

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-pending-oidc-review]",
  )) {
    button.addEventListener("click", () => {
      const pendingID = button.dataset.pendingId;
      if (!pendingID) return;

      name.textContent = button.dataset.pendingName || "Review OIDC identity";
      const username = button.dataset.pendingUsername || "";
      const email = button.dataset.pendingEmail || "";
      profile.textContent = email ? `${username} · ${email}` : username;
      issuer.textContent = button.dataset.pendingIssuer || "";
      subject.textContent = button.dataset.pendingSubject || "";

      const base = `/admin/oidc/pending/${encodeURIComponent(pendingID)}`;
      linkForm.action = `${base}/link`;
      approveForm.action = `${base}/approve`;
      rejectForm.action = `${base}/reject`;

      const suggestedID = button.dataset.suggestedUserId || "";
      user.value = [...user.options].some(
        (option) => option.value === suggestedID,
      )
        ? suggestedID
        : "";
      if (match) {
        const suggestedName = button.dataset.suggestedUserName || "";
        match.hidden = !suggestedID;
        match.textContent = suggestedName
          ? `Possible match: ${suggestedName}`
          : "Possible matching Lore account found.";
      }

      dialog.showModal();
      requestAnimationFrame(() => user.focus());
    });
  }

  for (const button of dialog.querySelectorAll<HTMLButtonElement>(
    "[data-pending-oidc-close]",
  )) {
    button.addEventListener("click", () => dialog.close());
  }
  dialog.addEventListener("click", (event: MouseEvent) => {
    if (event.target === dialog) dialog.close();
  });
}

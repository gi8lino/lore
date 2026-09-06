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
  const accountEnabled = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-enabled]",
  );
  const localLogin = dialog.querySelector<HTMLElement>(
    "[data-admin-user-local-login]",
  );
  const localCredentialEnabled = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-local-credential-enabled]",
  );
  const localCredentialUpdate = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-local-credential-update]",
  );
  const localPassword = dialog.querySelector<HTMLDetailsElement>(
    "[data-admin-user-local-password]",
  );
  const localPasswordInput = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-local-password-input]",
  );
  const localPasswordConfirm = dialog.querySelector<HTMLInputElement>(
    "[data-admin-user-local-password-confirm]",
  );
  const groupInputs = [
    ...dialog.querySelectorAll<HTMLInputElement>('input[name="group_id"]'),
  ];
  if (!form || !name || !identity || !role) return;

  const editorForm = form;
  const editorName = name;
  const editorIdentity = identity;
  const editorRole = role;

  localPassword?.addEventListener("toggle", () => {
    for (const input of [localPasswordInput, localPasswordConfirm]) {
      if (!input) continue;

      input.disabled = !localPassword.open;

      if (!localPassword.open) input.value = "";
    }
  });

  function openUserEditor(button: HTMLButtonElement): void {
    const userID = button.dataset.userId;
    if (!userID) return;

    const groups = selectedGroupIDs(button);

    editorForm.reset();
    editorForm.action = `/admin/users/${encodeURIComponent(userID)}`;
    editorName.textContent =
      button.dataset.userName || button.dataset.userUsername || "Edit user";

    const username = button.dataset.userUsername || "";
    const email = button.dataset.userEmail || "";

    editorIdentity.textContent = email ? `${username} · ${email}` : username;
    editorRole.value = button.dataset.userRole || "viewer";

    if (accountEnabled) {
      accountEnabled.checked = button.dataset.userEnabled === "true";
    }
    if (localLogin && localCredentialEnabled && localCredentialUpdate) {
      const hasLocalCredential =
        button.dataset.userHasLocalCredential === "true";
      const canManageLocalCredential =
        hasLocalCredential && dialog.dataset.externalAuthActive === "true";

      localLogin.hidden = !canManageLocalCredential;
      localCredentialEnabled.checked =
        button.dataset.userLocalCredentialEnabled === "true";
      localCredentialEnabled.disabled = !canManageLocalCredential;
      localCredentialUpdate.disabled = !canManageLocalCredential;
    }
    if (localPassword && localPasswordInput && localPasswordConfirm) {
      localPassword.open = false;
      localPasswordInput.disabled = true;
      localPasswordConfirm.disabled = true;
      localPasswordInput.value = "";
      localPasswordConfirm.value = "";
    }

    for (const input of groupInputs) input.checked = groups.has(input.value);

    dialog.showModal();
    requestAnimationFrame(() => editorRole.focus());
  }

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-admin-user-edit]",
  )) {
    button.addEventListener("click", () => openUserEditor(button));
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

  const identityName = name;
  const identityProfile = profile;
  const identityIssuer = issuer;
  const identitySubject = subject;
  const identityUser = user;
  const identityLinkForm = linkForm;
  const identityApproveForm = approveForm;
  const identityRejectForm = rejectForm;

  function openPendingIdentityEditor(button: HTMLButtonElement): void {
    const pendingID = button.dataset.pendingId;
    if (!pendingID) return;

    identityName.textContent =
      button.dataset.pendingName || "Review OIDC identity";

    const username = button.dataset.pendingUsername || "";
    const email = button.dataset.pendingEmail || "";

    identityProfile.textContent = email ? `${username} · ${email}` : username;
    identityIssuer.textContent = button.dataset.pendingIssuer || "";
    identitySubject.textContent = button.dataset.pendingSubject || "";

    const base = `/admin/oidc/pending/${encodeURIComponent(pendingID)}`;

    identityLinkForm.action = `${base}/link`;
    identityApproveForm.action = `${base}/approve`;
    identityRejectForm.action = `${base}/reject`;

    const suggestedID = button.dataset.suggestedUserId || "";

    identityUser.value = [...identityUser.options].some(
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
    requestAnimationFrame(() => identityUser.focus());
  }

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-pending-oidc-review]",
  )) {
    button.addEventListener("click", () => openPendingIdentityEditor(button));
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

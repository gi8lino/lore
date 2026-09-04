// Rendered page interactions and page-local state.

export function initPage(): void {
  const pageHeading = document.querySelector<HTMLElement>(".page-heading");
  const pageReading = document.querySelector<HTMLElement>(".page-reading");
  const pageContents = document.querySelector<HTMLElement>(
    "[data-page-contents]",
  );
  const pageContentsToggle = document.querySelector<HTMLButtonElement>(
    "[data-page-contents-toggle]",
  );
  const pageContentsBackdrop = document.querySelector<HTMLButtonElement>(
    "[data-page-contents-backdrop]",
  );
  const pageContentsClose = document.querySelector<HTMLButtonElement>(
    "[data-page-contents-close]",
  );
  const mobileContents = window.matchMedia("(max-width: 800px)");
  let desktopContentsVisible =
    pageReading?.classList.contains("with-contents") ?? false;
  let scheduleActiveHeadingUpdate: () => void = () => {};

  function updatePageHeadingHeight(): void {
    const sticky = window.matchMedia("(min-width: 801px)").matches;
    const height = sticky && pageHeading ? pageHeading.offsetHeight : 0;
    document.documentElement.style.setProperty(
      "--page-heading-height",
      `${height}px`,
    );
  }

  updatePageHeadingHeight();
  window.addEventListener("resize", updatePageHeadingHeight);
  if (pageHeading && "ResizeObserver" in window)
    new ResizeObserver(updatePageHeadingHeight).observe(pageHeading);

  if (pageContents) {
    const contents = pageContents;
    const entries = [
      ...contents.querySelectorAll<HTMLAnchorElement>('a[href^="#"]'),
    ]
      .map((link) => {
        const heading = document.getElementById(
          decodeURIComponent(link.hash.slice(1)),
        );
        return heading ? { heading, link } : null;
      })
      .filter(
        (entry): entry is { heading: HTMLElement; link: HTMLAnchorElement } =>
          entry !== null,
      );

    let activeLink: HTMLAnchorElement | null = null;
    let scheduled = false;

    function revealActiveLink(link: HTMLAnchorElement): void {
      if (contents.offsetParent === null) return;
      const container = contents.getBoundingClientRect();
      const item = link.getBoundingClientRect();
      if (item.top < container.top)
        contents.scrollTop -= container.top - item.top;
      else if (item.bottom > container.bottom)
        contents.scrollTop += item.bottom - container.bottom;
    }

    function setActiveHeading(link: HTMLAnchorElement): void {
      if (activeLink === link) return;
      activeLink?.classList.remove("active");
      activeLink?.removeAttribute("aria-current");
      link.classList.add("active");
      link.setAttribute("aria-current", "location");
      activeLink = link;
      revealActiveLink(link);
    }

    function updateActiveHeading(): void {
      scheduled = false;
      const first = entries[0];
      if (!first) return;
      const headingBottom = pageHeading?.getBoundingClientRect().bottom ?? 0;
      const threshold = Math.max(96, headingBottom + 12);
      let current = first;
      for (const entry of entries) {
        if (entry.heading.getBoundingClientRect().top <= threshold)
          current = entry;
        else break;
      }
      if (
        window.innerHeight + window.scrollY >=
        document.documentElement.scrollHeight - 2
      ) {
        current = entries.at(-1) ?? current;
      }
      setActiveHeading(current.link);
    }

    scheduleActiveHeadingUpdate = () => {
      if (scheduled) return;
      scheduled = true;
      requestAnimationFrame(updateActiveHeading);
    };

    window.addEventListener("scroll", scheduleActiveHeadingUpdate, {
      passive: true,
    });
    window.addEventListener("resize", scheduleActiveHeadingUpdate);
    updateActiveHeading();
  }

  async function persistPageContentsPreference(show: boolean): Promise<void> {
    const url = pageContentsToggle?.dataset.preferenceUrl;
    if (!url) return;
    const response = await fetch(url, {
      method: "POST",
      headers: {
        Accept: "text/plain",
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
      },
      body: new URLSearchParams({ show: String(show) }),
    });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
  }

  function setPageContentsVisible(show: boolean): void {
    if (!pageReading || !pageContentsToggle) return;
    desktopContentsVisible = show;
    pageReading.classList.toggle("with-contents", show);
    pageReading.classList.toggle("contents-hidden", !show);
    pageContentsToggle.setAttribute("aria-pressed", String(show));
    const label = show ? "Hide page contents" : "Show page contents";
    pageContentsToggle.setAttribute("aria-label", label);
    pageContentsToggle.title = label;
    if (show) scheduleActiveHeadingUpdate();
  }

  function setContentsPopoverOpen(open: boolean, restoreFocus = false): void {
    if (!pageReading || !pageContents || !pageContentsToggle) return;
    const nextOpen = open && mobileContents.matches;
    pageReading.classList.toggle("contents-popover-open", nextOpen);
    document.body.classList.toggle("contents-popover-open", nextOpen);
    if (pageContentsBackdrop) pageContentsBackdrop.hidden = !nextOpen;
    pageContentsToggle.setAttribute("aria-expanded", String(nextOpen));
    pageContents.setAttribute("aria-hidden", String(!nextOpen));
    const label = nextOpen ? "Hide page contents" : "Show page contents";
    pageContentsToggle.setAttribute("aria-label", label);
    pageContentsToggle.title = label;
    if (nextOpen) pageContentsClose?.focus();
    else if (restoreFocus) pageContentsToggle.focus();
  }

  function syncContentsMode(): void {
    setContentsPopoverOpen(false);
    if (!pageContents || !pageContentsToggle) return;
    if (mobileContents.matches) {
      pageContentsToggle.removeAttribute("aria-pressed");
      pageContents.setAttribute("aria-hidden", "true");
      return;
    }
    pageContents.removeAttribute("aria-hidden");
    pageContentsToggle.setAttribute(
      "aria-pressed",
      String(desktopContentsVisible),
    );
    const label = desktopContentsVisible
      ? "Hide page contents"
      : "Show page contents";
    pageContentsToggle.setAttribute("aria-label", label);
    pageContentsToggle.title = label;
  }

  pageContentsToggle?.addEventListener("click", async () => {
    if (!pageReading || !pageContentsToggle) return;
    if (mobileContents.matches) {
      setContentsPopoverOpen(
        !pageReading.classList.contains("contents-popover-open"),
      );
      return;
    }
    const previous = pageReading.classList.contains("with-contents");
    const next = !previous;
    setPageContentsVisible(next);
    pageContentsToggle.disabled = true;
    try {
      await persistPageContentsPreference(next);
    } catch (error) {
      console.error("failed to save page contents preference", error);
      setPageContentsVisible(previous);
    } finally {
      pageContentsToggle.disabled = false;
    }
  });
  pageContentsBackdrop?.addEventListener("click", () =>
    setContentsPopoverOpen(false, true),
  );
  pageContentsClose?.addEventListener("click", () =>
    setContentsPopoverOpen(false, true),
  );
  pageContents?.addEventListener("click", (event: MouseEvent) => {
    const target = event.target;
    if (
      mobileContents.matches &&
      target instanceof Element &&
      target.closest('a[href^="#"]')
    )
      setContentsPopoverOpen(false);
  });
  document.addEventListener("keydown", (event: KeyboardEvent) => {
    if (
      event.key !== "Escape" ||
      !pageReading?.classList.contains("contents-popover-open")
    )
      return;
    event.preventDefault();
    setContentsPopoverOpen(false, true);
  });
  mobileContents.addEventListener("change", syncContentsMode);
  syncContentsMode();

  const revisionDialog = document.querySelector<HTMLDialogElement>(
    "[data-revision-dialog]",
  );
  const revisionDialogOpen = document.querySelector<HTMLButtonElement>(
    "[data-revision-dialog-open]",
  );
  const revisionDialogClose = revisionDialog?.querySelector<HTMLButtonElement>(
    "[data-revision-dialog-close]",
  );
  const revisionDialogBody = revisionDialog?.querySelector<HTMLElement>(
    "[data-revision-dialog-body]",
  );
  let revisionHistoryLoaded = false;

  async function loadRevisionHistory(): Promise<void> {
    if (revisionHistoryLoaded || !revisionDialogOpen || !revisionDialogBody)
      return;
    const revisionURL = revisionDialogOpen.dataset.revisionUrl;
    if (!revisionURL) return;
    revisionDialogOpen.disabled = true;
    revisionDialogBody.innerHTML =
      '<p class="muted">Loading revision history…</p>';
    try {
      const response = await fetch(revisionURL, {
        headers: { Accept: "text/html" },
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      revisionDialogBody.innerHTML = await response.text();
      revisionHistoryLoaded = true;
    } catch (error) {
      console.error("failed to load revision history", error);
      revisionDialogBody.innerHTML =
        '<p class="revision-dialog-error">Revision history could not be loaded. Close the dialog and try again.</p>';
    } finally {
      revisionDialogOpen.disabled = false;
    }
  }

  revisionDialogOpen?.addEventListener("click", () => {
    revisionDialog?.showModal();
    void loadRevisionHistory();
  });
  revisionDialogClose?.addEventListener("click", () => revisionDialog?.close());
  revisionDialog?.addEventListener("click", (event) => {
    if (event.target === revisionDialog) revisionDialog.close();
  });

  const moveDialog = document.querySelector<HTMLDialogElement>(
    "[data-move-page-dialog]",
  );
  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-move-page-open]",
  )) {
    button.addEventListener("click", () => moveDialog?.showModal());
  }
  for (const button of moveDialog?.querySelectorAll<HTMLButtonElement>(
    "[data-move-page-close]",
  ) || []) {
    button.addEventListener("click", () => moveDialog?.close());
  }
  moveDialog?.addEventListener("click", (event) => {
    if (event.target === moveDialog) moveDialog.close();
  });

  const commentDialog = document.querySelector<HTMLDialogElement>(
    "[data-comment-dialog]",
  );
  const commentAnchor = commentDialog?.querySelector<HTMLTextAreaElement>(
    "[data-comment-anchor]",
  );

  function selectedPageText(): string {
    const selection = window.getSelection();
    if (!selection || selection.isCollapsed || !selection.rangeCount) return "";
    const range = selection.getRangeAt(0);
    const prose = document.querySelector<HTMLElement>(".page-reading .prose");
    if (!prose || !prose.contains(range.commonAncestorContainer)) return "";
    return selection.toString().trim().slice(0, 500);
  }

  for (const button of document.querySelectorAll<HTMLButtonElement>(
    "[data-comment-dialog-open]",
  )) {
    button.addEventListener("click", () => {
      if (commentAnchor && !commentAnchor.value.trim())
        commentAnchor.value = selectedPageText();
      commentDialog?.showModal();
      requestAnimationFrame(() =>
        commentDialog
          ?.querySelector<HTMLTextAreaElement>('textarea[name="body"]')
          ?.focus(),
      );
    });
  }
  for (const button of commentDialog?.querySelectorAll<HTMLButtonElement>(
    "[data-comment-dialog-close]",
  ) || []) {
    button.addEventListener("click", () => commentDialog?.close());
  }
  commentDialog?.addEventListener("click", (event) => {
    if (event.target === commentDialog) commentDialog.close();
  });
}

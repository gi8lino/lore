// Notification inbox interactions.

async function markRead(id: string): Promise<void> {
  const response = await fetch(`/api/notifications/${id}/read`, {
    method: "POST",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
}

// Initializes notifications.
export function initNotifications(): void {
  const menu = document.querySelector<HTMLElement>("[data-notification-menu]");
  if (!menu) return;

  menu.addEventListener("click", (event: MouseEvent) => {
    const target = event.target;
    if (!(target instanceof Element)) return;

    const item = target.closest<HTMLElement>("[data-notification-id]");
    if (!item || !item.classList.contains("unread")) return;

    const id = item.dataset.notificationId;
    if (!id) return;

    void markRead(id)
      .then(() => {
        item.classList.remove("unread");

        const badge = menu.querySelector<HTMLElement>(".notification-badge");
        if (!badge) return;

        const next = Math.max(0, Number(badge.textContent || 0) - 1);

        if (next === 0) badge.remove();
        else badge.textContent = String(next);
      })
      .catch(() => {});
  });

  menu
    .querySelector<HTMLElement>("[data-notifications-read-all]")
    ?.addEventListener("click", (event: MouseEvent) => {
      event.preventDefault();

      const trigger = event.currentTarget;
      if (!(trigger instanceof HTMLElement)) return;

      void markRead("all")
        .then(() => {
          for (const item of menu.querySelectorAll<HTMLElement>(
            ".notification-item.unread",
          ))
            item.classList.remove("unread");

          menu.querySelector<HTMLElement>(".notification-badge")?.remove();
          trigger.remove();
        })
        .catch(() => {});
    });
}

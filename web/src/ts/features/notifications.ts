// Notification inbox interactions.

import { requiredAttribute } from "../core/dom.ts";
import { requestJSON } from "../core/http.ts";

async function markRead(id: string): Promise<void> {
  await requestJSON(`/api/notifications/${id}/read`, { method: "POST" });
}

function decrementNotificationBadge(menu: HTMLElement): void {
  const badge = menu.querySelector<HTMLElement>(".notification-badge");
  if (!badge) return;

  const next = Math.max(0, Number(badge.textContent || 0) - 1);
  if (next === 0) {
    badge.remove();
    return;
  }

  badge.textContent = String(next);
}

async function readNotification(
  menu: HTMLElement,
  item: HTMLElement,
  id: string,
): Promise<void> {
  try {
    await markRead(id);
  } catch {
    return;
  }

  item.classList.remove("unread");
  decrementNotificationBadge(menu);
}

function handleNotificationClick(menu: HTMLElement, event: MouseEvent): void {
  const target = event.target;
  if (!(target instanceof Element)) return;

  const item = target.closest<HTMLElement>("[data-notification-id]");
  if (!item || !item.classList.contains("unread")) return;

  const id = requiredAttribute(item, "data-notification-id");

  void readNotification(menu, item, id);
}

async function readAllNotifications(
  menu: HTMLElement,
  trigger: HTMLElement,
): Promise<void> {
  try {
    await markRead("all");
  } catch {
    return;
  }

  for (const item of menu.querySelectorAll<HTMLElement>(
    ".notification-item.unread",
  )) {
    item.classList.remove("unread");
  }

  menu.querySelector<HTMLElement>(".notification-badge")?.remove();
  trigger.remove();
}

function handleReadAllClick(menu: HTMLElement, event: MouseEvent): void {
  event.preventDefault();

  const trigger = event.currentTarget;
  if (!(trigger instanceof HTMLElement)) return;

  void readAllNotifications(menu, trigger);
}

// Initializes notifications.
export function initNotifications(): void {
  const menu = document.querySelector<HTMLElement>("[data-notification-menu]");
  if (!menu) return;

  menu.addEventListener("click", (event: MouseEvent) =>
    handleNotificationClick(menu, event),
  );

  menu
    .querySelector<HTMLElement>("[data-notifications-read-all]")
    ?.addEventListener("click", (event: MouseEvent) =>
      handleReadAllClick(menu, event),
    );
}

// Service worker setup and private-cache protection.

const pageCachePrefix = "lore-pages-";

// Clears private page caches.
async function clearPrivatePageCaches(): Promise<void> {
  if (!("caches" in window)) return;

  const names = await caches.keys();

  await Promise.all(
    names
      .filter((name) => name.startsWith(pageCachePrefix))
      .map((name) => caches.delete(name)),
  );
}

// Configures worker.
function configureWorker(registration: ServiceWorkerRegistration): void {
  const userID = document.body?.dataset.userId || "";
  const worker =
    registration.active || registration.waiting || registration.installing;

  worker?.postMessage({ type: "configure-user", userID });
}

// Protects logout.
function protectLogout(): void {
  const form = document.querySelector<HTMLFormElement>(
    'form[action="/auth/logout"]',
  );
  if (!form || !("caches" in window)) return;

  form.addEventListener("submit", async (event: SubmitEvent) => {
    if (form.dataset.pwaLogout === "true") return;

    event.preventDefault();

    try {
      navigator.serviceWorker?.controller?.postMessage({
        type: "configure-user",
        userID: "",
      });
      await clearPrivatePageCaches();
    } catch {
      /* best effort */
    }

    form.dataset.pwaLogout = "true";
    form.submit();
  });
}

// Initializes pwa.
export function initPWA(): void {
  protectLogout();
  if (
    !("serviceWorker" in navigator) ||
    (location.protocol === "http:" &&
      location.hostname !== "localhost" &&
      location.hostname !== "127.0.0.1")
  )
    return;

  async function registerServiceWorker(): Promise<void> {
    try {
      await navigator.serviceWorker.register("/sw.js", { scope: "/" });
      await configureWorker(await navigator.serviceWorker.ready);
    } catch {
      // Offline support is best effort and must never interfere with normal navigation.
    }
  }

  window.addEventListener("load", () => void registerServiceWorker());
}

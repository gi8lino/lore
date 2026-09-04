/// <reference lib="webworker" />

// Offline asset caching and user-scoped private page caching.

const assetCacheName = "lore-assets-v1";
const pageCachePrefix = "lore-pages-v1-";
let pageCacheName = "";

const serviceWorker = self as unknown as ServiceWorkerGlobalScope;

type ConfigureUserMessage = {
  type?: unknown;
  userID?: unknown;
};

serviceWorker.addEventListener("install", () => {
  void serviceWorker.skipWaiting();
});

serviceWorker.addEventListener("activate", (event: ExtendableEvent) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter(
              (name) =>
                name.startsWith("lore-offline-") ||
                (name.startsWith("lore-assets-") && name !== assetCacheName),
            )
            .map((name) => caches.delete(name)),
        ),
      )
      .then(() => serviceWorker.clients.claim()),
  );
});

serviceWorker.addEventListener("message", (event: ExtendableMessageEvent) => {
  const data = event.data as ConfigureUserMessage | null;
  if (data?.type !== "configure-user") return;

  const userID = String(data.userID || "").replace(/[^0-9A-Za-z_-]/g, "");
  if (!userID) {
    pageCacheName = "";
    return;
  }

  pageCacheName = `${pageCachePrefix}${userID}`;
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter(
              (name) =>
                name.startsWith(pageCachePrefix) && name !== pageCacheName,
            )
            .map((name) => caches.delete(name)),
        ),
      )
      .then(() => undefined),
  );
});

function cacheablePage(url: URL): boolean {
  return (
    url.origin === serviceWorker.location.origin &&
    (url.pathname === "/" || url.pathname.startsWith("/pages/"))
  );
}

serviceWorker.addEventListener("fetch", (event: FetchEvent) => {
  const request = event.request;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== serviceWorker.location.origin) return;

  if (url.pathname.startsWith("/assets/")) {
    event.respondWith(
      caches.match(request).then(
        (cached) =>
          cached ||
          fetch(request).then((response) => {
            if (response.ok) {
              void caches
                .open(assetCacheName)
                .then((cache) => cache.put(request, response.clone()));
            }
            return response;
          }),
      ),
    );
    return;
  }

  if (!pageCacheName || !cacheablePage(url)) return;
  event.respondWith(
    fetch(request)
      .then((response) => {
        if (response.ok) {
          void caches
            .open(pageCacheName)
            .then((cache) => cache.put(request, response.clone()));
        }
        return response;
      })
      .catch(() =>
        caches
          .open(pageCacheName)
          .then((cache) => cache.match(request))
          .then((cached) => cached || Response.error()),
      ),
  );
});

/// <reference lib="webworker" />

// Offline asset caching and user-scoped private page caching.

const assetCacheName = "lore-assets-v1";
const pageCachePrefix = "lore-pages-v2-";
let pageCacheName = "";
const clientCaches = new Map<string, string>();
let generation = 0;
let privateOperations: Promise<unknown> = Promise.resolve();

// Serialize writes and deletion so logout cannot be undone by an in-flight put.
function privateOperation(work: () => Promise<unknown>): Promise<void> {
  const result = privateOperations.then(work).then(() => undefined);
  privateOperations = result.catch(() => undefined);
  return result;
}

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
                (name.startsWith("lore-pages-") &&
                  !name.startsWith(pageCachePrefix)) ||
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

  const source = event.source;
  if (!source || !("id" in source)) return;
  const userID = String(data.userID || "");
  const nextCache = /^[1-9][0-9]*$/.test(userID)
    ? `${pageCachePrefix}${userID}`
    : "";
  if (nextCache !== pageCacheName || !nextCache) {
    generation++;
    clientCaches.clear();
    pageCacheName = nextCache;
  }
  if (nextCache) clientCaches.set(source.id, nextCache);
  event.waitUntil(
    privateOperation(async () => {
      const names = await caches.keys();
      await Promise.all(
        names
          .filter(
            (name) =>
              name.startsWith(pageCachePrefix) && name !== pageCacheName,
          )
          .map((name) => caches.delete(name)),
      );
    }),
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
      // Use HTTP cache policy online so unversioned assets revalidate between releases.
      fetch(request).then(
        async (response) => {
          if (response.ok && !response.redirected) {
            const copy = response.clone();
            await caches
              .open(assetCacheName)
              .then((cache) => cache.put(request, copy))
              .catch(() => undefined);
          }
          return response;
        },
        async () => {
          const cache = await caches.open(assetCacheName);
          return (await cache.match(request)) || Response.error();
        },
      ),
    );
    return;
  }

  const requestCache = clientCaches.get(event.clientId);
  if (!requestCache || !cacheablePage(url)) return;
  const requestGeneration = generation;
  const stillCurrent = () =>
    requestGeneration === generation && requestCache === pageCacheName;

  event.respondWith(
    fetch(request).then(
      async (response) => {
        // Redirected login pages and responses for a different signed-in user
        // must never enter the requesting tab's private cache.
        const responseCache = `${pageCachePrefix}${response.headers.get("X-Lore-User-ID") || ""}`;
        if (
          response.ok &&
          !response.redirected &&
          responseCache === requestCache &&
          stillCurrent()
        ) {
          const copy = response.clone();
          await privateOperation(async () => {
            if (!stillCurrent()) return;
            const cache = await caches.open(requestCache);
            await cache.put(request, copy);
          }).catch(() => undefined);
        }
        return response;
      },
      async () => {
        if (!stillCurrent()) return Response.error();
        const cache = await caches.open(requestCache);
        const cached = await cache.match(request);
        return stillCurrent() ? cached || Response.error() : Response.error();
      },
    ),
  );
});

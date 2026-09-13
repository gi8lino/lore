// Only core code runs in Lore's document. Browser modules receive their own
// marked block's text inside a sandboxed frame and cannot return host HTML.
type Root = Document | HTMLElement;
type Module = {
  plugin_id: string;
  module_id: string;
  digest: string;
  name: string;
  frame_url: string;
};
type Active = {
  frame: HTMLIFrameElement;
  source: HTMLElement;
  version: string;
  finish: () => void;
  ready: Promise<void>;
  cleanup: () => void;
};
const active = new Map<HTMLElement, Active>();
let loading: Promise<Module[]> | undefined;
let watching = false;
const identifier = /^[a-z0-9][a-z0-9._-]{0,127}$/;
function validModule(value: unknown): value is Module {
  if (typeof value !== "object" || value === null) return false;
  const m = value as Record<string, unknown>;
  return (
    typeof m.plugin_id === "string" &&
    identifier.test(m.plugin_id) &&
    typeof m.module_id === "string" &&
    identifier.test(m.module_id) &&
    typeof m.digest === "string" &&
    /^[a-f0-9]{64}$/.test(m.digest) &&
    typeof m.name === "string" &&
    typeof m.frame_url === "string"
  );
}
function modules(): Promise<Module[]> {
  if (loading) return loading;
  loading = (async () => {
    const endpoint = new URL(
      document.body.dataset.pluginModules || "/plugins/modules.json",
      location.href,
    );
    if (endpoint.origin !== location.origin)
      throw new Error("Invalid plugin catalog origin");
    const response = await fetch(endpoint, {
      cache: "no-store",
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error("Plugin catalog unavailable");
    const data: unknown = await response.json();
    if (!Array.isArray(data) || !data.every(validModule))
      throw new Error("Invalid plugin catalog");
    return data.filter((m) => {
      const frame = new URL(m.frame_url, location.href);
      return (
        frame.origin === location.origin &&
        frame.pathname.startsWith(new URL("./", endpoint).pathname) &&
        !frame.search &&
        !frame.hash
      );
    });
  })().finally(() => {
    loading = undefined;
  });
  return loading;
}
function remove(block: HTMLElement): void {
  const state = active.get(block);
  if (!state) return;
  state.cleanup();
  state.frame.remove();
  state.source.hidden = false;
  state.finish();
  active.delete(block);
}
function mount(block: HTMLElement, module: Module): Promise<void> {
  const source = block.querySelector<HTMLElement>(":scope > pre");
  if (!source || (source.textContent?.length || 0) > 1_000_000)
    return Promise.resolve();
  const frame = document.createElement("iframe");
  frame.className = "lore-plugin-frame";
  frame.title = module.name;
  frame.setAttribute("sandbox", "allow-scripts");
  frame.setAttribute("referrerpolicy", "no-referrer");
  frame.setAttribute(
    "allow",
    "camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'",
  );
  frame.src = module.frame_url;
  const token = Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");
  let finish!: () => void;
  const ready = new Promise<void>((resolve) => {
    finish = resolve;
  });
  const timeout = window.setTimeout(() => remove(block), 15000);
  const message = (event: MessageEvent<unknown>) => {
    if (
      event.source !== frame.contentWindow ||
      typeof event.data !== "object" ||
      event.data === null
    )
      return;
    const data = event.data as Record<string, unknown>;
    if (data.type === "lore-plugin-listening") {
      frame.contentWindow?.postMessage(
        {
          type: "lore-plugin-render",
          token,
          source: source.textContent || "",
          theme: document.documentElement.style.colorScheme,
        },
        "*",
      );
      return;
    }
    if (data.token !== token) return;
    if (
      data.type === "lore-plugin-ready" &&
      typeof data.height === "number" &&
      Number.isFinite(data.height)
    ) {
      frame.style.height = `${Math.min(10000, Math.max(24, data.height))}px`;
      source.hidden = true;
      frame.dataset.pluginReady = "true";
      clearTimeout(timeout);
      finish();
    } else if (data.type === "lore-plugin-error") {
      remove(block);
    }
  };
  window.addEventListener("message", message);
  active.set(block, {
    frame,
    source,
    ready,
    finish,
    version: module.digest,
    cleanup: () => {
      clearTimeout(timeout);
      window.removeEventListener("message", message);
    },
  });
  block.append(frame);
  return ready;
}
function watch(): void {
  if (watching) return;
  watching = true;
  new MutationObserver(() => {
    for (const block of active.keys()) if (!block.isConnected) remove(block);
  }).observe(document.body, { childList: true, subtree: true });
  if (document.body.dataset.pluginLive !== "false") {
    window.setInterval(() => {
      void renderPluginModules().catch(() => {
        for (const block of active.keys()) remove(block);
      });
    }, 3000);
  }
}
export async function renderPluginModules(
  root: Root = document,
): Promise<void> {
  const blocks = [
    ...root.querySelectorAll<HTMLElement>(
      "[data-lore-plugin][data-lore-module]",
    ),
  ];
  if (!blocks.length && !active.size) return;
  watch();
  const enabled = await modules().catch(() => [] as Module[]);
  const find = (block: HTMLElement) =>
    enabled.find(
      (m) =>
        m.plugin_id === block.dataset.lorePlugin &&
        m.module_id === block.dataset.loreModule,
    );
  for (const [block, state] of active)
    if (!block.isConnected || find(block)?.digest !== state.version)
      remove(block);
  await Promise.all(
    blocks.map((block) => {
      const module = find(block);
      if (!module) return Promise.resolve();
      return active.get(block)?.ready || mount(block, module);
    }),
  );
}

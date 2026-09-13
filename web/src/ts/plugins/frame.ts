// Core-owned harness. Plugin code runs only in this opaque, resource-restricted
// document. The only outgoing protocol is readiness, errors and bounded height.
(() => {
  let started = false;
  window.addEventListener("message", async (event: MessageEvent<unknown>) => {
    if (
      started ||
      event.source !== parent ||
      typeof event.data !== "object" ||
      event.data === null
    )
      return;
    const input = event.data as Record<string, unknown>;
    if (
      input.type !== "lore-plugin-render" ||
      typeof input.token !== "string" ||
      typeof input.source !== "string" ||
      input.source.length > 1_000_000
    )
      return;
    started = true;
    const send = (type: string, height = 0) =>
      parent.postMessage({ type, token: input.token, height }, "*");
    try {
      const url = document.body.dataset.pluginJavascript;
      const root = document.getElementById("plugin-root");
      if (!url || !root) throw new Error("Missing module");
      await new Promise<void>((resolve, reject) => {
        const script = document.createElement("script");
        script.src = url;
        script.onload = () => resolve();
        script.onerror = () => reject(new Error("Module load failed"));
        document.head.append(script);
      });
      const module: unknown = (
        globalThis as unknown as { lorePlugin?: unknown }
      ).lorePlugin;
      if (
        typeof module !== "object" ||
        module === null ||
        !("render" in module) ||
        typeof module.render !== "function"
      )
        throw new Error("Invalid module");
      document.documentElement.style.colorScheme =
        input.theme === "dark" ? "dark" : "light";
      await module.render(root, {
        source: input.source,
        theme: input.theme === "dark" ? "dark" : "light",
      });
      const measure = () =>
        send(
          "lore-plugin-ready",
          Math.min(
            10000,
            Math.max(24, Math.ceil(root.getBoundingClientRect().height)),
          ),
        );
      measure();
      new ResizeObserver(measure).observe(root);
    } catch {
      send("lore-plugin-error");
    }
  });
  parent.postMessage({ type: "lore-plugin-listening" }, "*");
})();

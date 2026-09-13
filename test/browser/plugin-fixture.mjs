import { readFile } from "node:fs/promises";
export const plugin = {
  plugin_id: "io.lore.mermaid",
  module_id: "diagrams",
  name: "Mermaid",
  digest: "a".repeat(64),
};
export const base = `/plugins/${plugin.plugin_id}/${plugin.digest}/`;
export const block =
  '<div data-lore-plugin="io.lore.mermaid" data-lore-module="diagrams"><pre><code class="language-mermaid">graph LR; A --> B</code></pre></div>';
// Browser fixture uses actual package JS/CSS and the core harness. HTTP tests
// independently verify the generated frame and response policy.
export async function pluginRoute(
  route,
  { enabled = true, fake = false } = {},
) {
  const url = new URL(route.request().url());
  const path = url.pathname;
  if (!path.startsWith("/plugins/")) return false;
  if (path === "/plugins/modules.json") {
    await route.fulfill({
      json: enabled
        ? [{ ...plugin, frame_url: base + "frames/diagrams.html" }]
        : [],
    });
    return true;
  }
  if (!enabled) {
    await route.fulfill({ status: 404 });
    return true;
  }
  if (path === base + "frames/diagrams.html") {
    const policy = `default-src 'none'; script-src ${url.origin}${base}assets/ ${url.origin}/plugins/runtime.js; style-src 'unsafe-inline' ${url.origin}${base}assets/; img-src data:; font-src data:; connect-src 'none'; sandbox allow-scripts`;
    await route.fulfill({
      contentType: "text/html",
      headers: { "Content-Security-Policy": policy },
      body: `<html><head><link rel="stylesheet" href="${base}assets/plugin.css"><script defer src="/plugins/runtime.js"></script></head><body data-plugin-javascript="${base}assets/plugin.js"><main id="plugin-root"></main></body></html>`,
    });
  } else if (path === "/plugins/runtime.js") {
    await route.fulfill({
      contentType: "text/javascript",
      body: await readFile(
        new URL("../../web/dist/js/plugins/frame.js", import.meta.url),
      ),
    });
  } else if (path === base + "assets/plugin.js" && fake) {
    await route.fulfill({
      contentType: "text/javascript",
      body: `globalThis.lorePlugin={async render(root){await new Promise(r=>setTimeout(r,150));root.innerHTML='<svg height="80" aria-label="Diagram"></svg>';}};`,
    });
  } else if (path.startsWith(base + "assets/")) {
    const name = path.slice((base + "assets/").length);
    if (!["plugin.js", "plugin.css", "mermaid.min.js"].includes(name)) {
      await route.fulfill({ status: 404 });
      return true;
    }
    await route.fulfill({
      contentType: name.endsWith(".css") ? "text/css" : "text/javascript",
      body: await readFile(
        new URL("../../plugins/mermaid/assets/" + name, import.meta.url),
      ),
    });
  } else {
    await route.fulfill({ status: 404 });
  }
  return true;
}

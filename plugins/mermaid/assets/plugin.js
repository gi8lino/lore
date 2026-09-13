const assetBase = document.currentScript.src;
async function load() {
  if (globalThis.mermaid) return;
  await new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = new URL("./mermaid.min.js", assetBase).href;
    script.onload = resolve;
    script.onerror = () => reject(new Error("Mermaid could not be loaded"));
    document.head.append(script);
  });
}

// All DOM work is confined to the opaque sandbox. The host supplies only this
// diagram's text and presentation preferences, never page DOM or credentials.
globalThis.lorePlugin = {
  async render(root, context) {
    await load();
    const diagram = document.createElement("div");
    diagram.className = "mermaid";
    diagram.textContent = context.source;
    root.replaceChildren(diagram);
    globalThis.mermaid.initialize({
      startOnLoad: false,
      theme: context.theme === "dark" ? "dark" : "default",
      securityLevel: "strict",
    });
    await globalThis.mermaid.run({ nodes: [diagram] });
  },
};

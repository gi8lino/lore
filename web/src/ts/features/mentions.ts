// User mention detection and autocomplete.

const resultLimit = 50;

type Fence = { character: string; length: number };
export type MentionTrigger = { start: number; query: string };
type MentionUser = {
  username: string;
  display_name?: string;
  role?: string;
  self?: boolean;
};
type CaretOffset = { left: number; top: number };

// Reports whether a caret offset is inside fenced code.
function fencedCodeAt(value: string, caret: number): boolean {
  const lines = value.slice(0, caret).split("\n");
  let fence: Fence | null = null;

  for (const line of lines) {
    const match = line.match(/^ {0,3}(`{3,}|~{3,})/u);
    if (!match) continue;

    const marker = match[1];
    const next: Fence = { character: marker[0], length: marker.length };
    if (!fence) {
      fence = next;
      continue;
    }

    if (fence.character === next.character && next.length >= fence.length)
      fence = null;
  }

  return fence !== null;
}

// Reports whether a caret offset is inside inline code.
function inlineCodeAt(value: string, caret: number): boolean {
  const lineStart = value.lastIndexOf("\n", Math.max(0, caret - 1)) + 1;
  const line = value.slice(lineStart, caret);
  let open = 0;

  for (let index = 0; index < line.length;) {
    if (line[index] !== "`" || (index > 0 && line[index - 1] === "\\")) {
      index += 1;
      continue;
    }

    let end = index + 1;

    while (line[end] === "`") end += 1;

    const length = end - index;

    if (open === 0) open = length;
    else if (length === open) open = 0;

    index = end;
  }

  return open !== 0;
}

// Finds an active user-mention trigger at the caret.
export function mentionTrigger(
  value: string,
  caret: number,
): MentionTrigger | null {
  if (fencedCodeAt(value, caret) || inlineCodeAt(value, caret)) return null;

  const lineStart = value.lastIndexOf("\n", Math.max(0, caret - 1)) + 1;
  const fragment = value.slice(lineStart, caret);
  const match = fragment.match(/(^|[\s([{>])@([A-Za-z0-9_.-]*)$/u);
  if (!match || match.index === undefined) return null;

  return {
    start: lineStart + match.index + match[1].length,
    query: match[2],
  };
}

// Builds the canonical mention text for a username.
export function mentionReplacement(username: string): string {
  return `@${username}`;
}

// Builds compact initials for a person label.
function initials(user: MentionUser): string {
  const value = (user.display_name || user.username || "?").trim();
  const parts = value.split(/\s+/u).filter(Boolean);
  if (!parts.length) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toLocaleUpperCase();

  return `${parts[0][0]}${parts.at(-1)?.[0] ?? ""}`.toLocaleUpperCase();
}

// Appends text with the matching query highlighted.
function appendHighlighted(
  target: HTMLElement,
  value: string,
  query: string,
): void {
  if (!query) {
    target.textContent = value;
    return;
  }

  const lower = value.toLocaleLowerCase();
  const start = lower.indexOf(query.toLocaleLowerCase());
  if (start < 0) {
    target.textContent = value;
    return;
  }

  target.append(document.createTextNode(value.slice(0, start)));

  const mark = document.createElement("mark");

  mark.textContent = value.slice(start, start + query.length);
  target.append(
    mark,
    document.createTextNode(value.slice(start + query.length)),
  );
}

// Calculates the caret offset inside a textarea.
function caretOffset(source: HTMLTextAreaElement, caret: number): CaretOffset {
  const computed = getComputedStyle(source);
  const mirror = document.createElement("div");
  const properties = [
    "font-family",
    "font-size",
    "font-weight",
    "font-style",
    "letter-spacing",
    "text-transform",
    "line-height",
    "padding-top",
    "padding-right",
    "padding-bottom",
    "padding-left",
    "border-top-width",
    "border-right-width",
    "border-bottom-width",
    "border-left-width",
  ];

  mirror.style.position = "fixed";
  mirror.style.left = "-10000px";
  mirror.style.top = "0";
  mirror.style.visibility = "hidden";
  mirror.style.whiteSpace = "pre-wrap";
  mirror.style.overflowWrap = "break-word";
  mirror.style.wordBreak = "normal";
  mirror.style.boxSizing = computed.boxSizing;
  mirror.style.width = `${source.offsetWidth}px`;

  for (const property of properties)
    mirror.style.setProperty(property, computed.getPropertyValue(property));

  mirror.textContent = source.value.slice(0, caret);

  const marker = document.createElement("span");

  marker.textContent = source.value.slice(caret, caret + 1) || "\u200b";
  mirror.append(marker);
  document.body.append(mirror);

  const lineHeight =
    Number.parseFloat(computed.lineHeight) ||
    Number.parseFloat(computed.fontSize) * 1.4;
  const offset = {
    left: marker.offsetLeft - source.scrollLeft,
    top: marker.offsetTop - source.scrollTop + lineHeight,
  };

  mirror.remove();
  return offset;
}

function mentionUsers(value: unknown): MentionUser[] {
  if (!Array.isArray(value)) return [];
  return value.filter(
    (item): item is MentionUser =>
      typeof item === "object" &&
      item !== null &&
      typeof (item as { username?: unknown }).username === "string",
  );
}

// Wires mention autocomplete behavior.
function setupMentionAutocomplete(source: HTMLTextAreaElement): void {
  const anchor = source.parentElement;
  if (!anchor) return;

  const suggestionAnchor = anchor;

  suggestionAnchor.classList.add("mention-suggestion-anchor");

  const menu = document.createElement("div");

  menu.className = "mention-suggestion-menu";
  menu.hidden = true;
  menu.setAttribute("role", "listbox");
  menu.setAttribute("aria-label", "Mention a user");
  suggestionAnchor.append(menu);

  let trigger: MentionTrigger | null = null;
  let results: MentionUser[] = [];
  let active = -1;
  let request = 0;

  function close(): void {
    request += 1;
    trigger = null;
    results = [];
    active = -1;
    menu.hidden = true;
    menu.replaceChildren();
    source.setAttribute("aria-expanded", "false");
  }

  function position(): void {
    if (menu.hidden) return;

    const sourceRect = source.getBoundingClientRect();
    const anchorRect = suggestionAnchor.getBoundingClientRect();
    const caret = caretOffset(source, source.selectionStart ?? 0);
    const menuWidth = Math.min(
      390,
      Math.max(260, suggestionAnchor.clientWidth - 16),
    );
    const rawLeft = sourceRect.left - anchorRect.left + caret.left;
    const maxLeft = Math.max(8, suggestionAnchor.clientWidth - menuWidth - 8);

    menu.style.width = `${menuWidth}px`;
    menu.style.left = `${Math.max(8, Math.min(rawLeft, maxLeft))}px`;
    menu.style.top = `${sourceRect.top - anchorRect.top + caret.top}px`;
  }

  function render(): void {
    menu.replaceChildren();

    if (!results.length) {
      const empty = document.createElement("div");

      empty.className = "mention-suggestion-empty";
      empty.textContent = trigger?.query
        ? "No matching people."
        : "No users available.";
      menu.append(empty);
    } else {
      for (const [index, user] of results.entries()) {
        const option = document.createElement("button");

        option.type = "button";
        option.className = "mention-suggestion-option";
        option.dataset.mentionIndex = String(index);
        option.setAttribute("role", "option");
        option.setAttribute("aria-selected", String(index === active));

        const avatar = document.createElement("span");

        avatar.className = "mention-suggestion-avatar";
        avatar.textContent = initials(user);

        const text = document.createElement("span");

        text.className = "mention-suggestion-text";

        const name = document.createElement("strong");

        appendHighlighted(
          name,
          user.display_name || user.username,
          trigger?.query || "",
        );

        const meta = document.createElement("small");

        appendHighlighted(meta, `@${user.username}`, trigger?.query || "");

        if (user.role) meta.append(document.createTextNode(` · ${user.role}`));
        if (user.self) meta.append(document.createTextNode(" · you"));

        text.append(name, meta);

        option.append(avatar, text);
        menu.append(option);
      }
    }

    menu.hidden = false;
    source.setAttribute("aria-expanded", "true");
    position();
  }

  function choose(index: number): void {
    const user = results[index];
    if (!user || !trigger) return;

    const end = source.selectionStart ?? trigger.start;

    source.setRangeText(
      mentionReplacement(user.username),
      trigger.start,
      end,
      "end",
    );
    source.dispatchEvent(new Event("input", { bubbles: true }));
    close();
    source.focus();
  }

  async function refresh(): Promise<void> {
    const next = mentionTrigger(source.value, source.selectionStart ?? 0);
    if (!next) {
      close();
      return;
    }

    trigger = next;

    const currentRequest = ++request;

    try {
      const response = await fetch(
        `/api/mentions/users?q=${encodeURIComponent(next.query)}`,
        {
          headers: { Accept: "application/json" },
        },
      );
      if (!response.ok || currentRequest !== request) return;

      results = mentionUsers(await response.json()).slice(0, resultLimit);
      active = results.length ? 0 : -1;
      render();
    } catch {
      if (currentRequest === request) close();
    }
  }

  source.addEventListener("input", () => void refresh());
  source.addEventListener("click", () => void refresh());
  source.addEventListener("scroll", position);
  source.addEventListener("keydown", (event: KeyboardEvent) => {
    if (menu.hidden) return;
    if (event.key === "ArrowDown" && results.length) {
      event.preventDefault();
      active = (active + 1) % results.length;
      render();
    } else if (event.key === "ArrowUp" && results.length) {
      event.preventDefault();
      active = (active - 1 + results.length) % results.length;
      render();
    } else if ((event.key === "Enter" || event.key === "Tab") && active >= 0) {
      event.preventDefault();
      choose(active);
    } else if (event.key === "Escape") {
      event.preventDefault();
      close();
    }
  });

  menu.addEventListener("mousedown", (event) => event.preventDefault());
  menu.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) return;

    const option = target.closest<HTMLElement>("[data-mention-index]");

    if (option) choose(Number(option.dataset.mentionIndex));
  });

  document.addEventListener("click", (event) => {
    const target = event.target;
    if (target !== source && target instanceof Node && !menu.contains(target))
      close();
  });
  window.addEventListener("resize", position);
}

// Initializes mention autocomplete.
export function initMentionAutocomplete(): void {
  for (const source of document.querySelectorAll<HTMLTextAreaElement>(
    "textarea[data-mention-autocomplete]",
  )) {
    setupMentionAutocomplete(source);
  }
}

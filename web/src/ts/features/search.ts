// Live global search behavior.

import { responseProblem } from "../core/http.ts";

interface SearchPage {
  slug: string;
  title: string;
}

function isSearchPage(value: unknown): value is SearchPage {
  if (typeof value !== "object" || value === null) return false;

  const page = value as Partial<SearchPage>;

  return typeof page.slug === "string" && typeof page.title === "string";
}

// Wires live search behavior.
function setupLiveSearch(form: HTMLFormElement): void {
  const input = form.querySelector<HTMLInputElement>('input[type="search"]');
  if (!input) return;

  const searchInput = input;

  const results = document.createElement("div");

  results.className = "search-suggestions";
  results.setAttribute("role", "listbox");
  results.hidden = true;
  form.append(results);

  let requestController: AbortController | null = null;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let activeIndex = -1;
  let resultLinks: HTMLAnchorElement[] = [];

  // Closes results.
  function closeResults(): void {
    results.hidden = true;
    searchInput.setAttribute("aria-expanded", "false");
    activeIndex = -1;
    resultLinks = [];
  }

  // Selects result.
  function selectResult(index: number): void {
    if (!resultLinks.length) return;

    activeIndex = Math.max(0, Math.min(index, resultLinks.length - 1));

    for (const [candidateIndex, link] of resultLinks.entries()) {
      link.classList.toggle("active", candidateIndex === activeIndex);
      link.setAttribute(
        "aria-selected",
        candidateIndex === activeIndex ? "true" : "false",
      );
    }

    resultLinks[activeIndex]?.scrollIntoView({ block: "nearest" });
  }

  // Renders results.
  function renderResults(pages: SearchPage[]): void {
    results.replaceChildren();
    activeIndex = -1;

    if (!pages.length) {
      const empty = document.createElement("div");

      empty.className = "search-suggestion-empty";
      empty.textContent = "No pages found.";
      results.append(empty);
      resultLinks = [];
    } else {
      resultLinks = pages.map((page) => {
        const link = document.createElement("a");

        link.className = "search-suggestion";
        link.href = `/pages/${page.slug}`;
        link.setAttribute("role", "option");
        link.setAttribute("aria-selected", "false");

        const title = document.createElement("strong");

        title.textContent = page.title;

        const path = document.createElement("small");

        path.textContent = page.slug;
        link.append(title, path);
        results.append(link);
        return link;
      });
    }

    results.hidden = false;
    searchInput.setAttribute("aria-expanded", "true");
  }

  // Runs the current live-search query.
  async function search(): Promise<void> {
    requestController?.abort();
    requestController = new AbortController();

    try {
      const response = await fetch(
        `/api/search?q=${encodeURIComponent(searchInput.value.trim())}`,
        {
          headers: { Accept: "application/json" },
          signal: requestController.signal,
        },
      );
      if (!response.ok) throw await responseProblem(response);

      const payload: unknown = await response.json();

      renderResults(Array.isArray(payload) ? payload.filter(isSearchPage) : []);
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") return;

      console.error("live search failed", error);
      closeResults();
    }
  }

  // Schedules search.
  function scheduleSearch(delay = 120): void {
    if (timer !== undefined) clearTimeout(timer);
    timer = setTimeout(() => void search(), delay);
  }

  searchInput.addEventListener("focus", () => scheduleSearch(0));
  searchInput.addEventListener("input", () => scheduleSearch());
  searchInput.addEventListener("keydown", (event: KeyboardEvent) => {
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        selectResult(activeIndex + 1);
        break;
      case "ArrowUp":
        event.preventDefault();
        selectResult(
          activeIndex <= 0 ? resultLinks.length - 1 : activeIndex - 1,
        );
        break;
      case "Enter":
        if (activeIndex >= 0) {
          event.preventDefault();
          resultLinks[activeIndex]?.click();
        }
        break;
      case "Escape":
        closeResults();
        searchInput.blur();
        break;
    }
  });

  document.addEventListener("pointerdown", (event: PointerEvent) => {
    if (event.target instanceof Node && !form.contains(event.target)) {
      closeResults();
    }
  });
}

// Initializes search.
export function initSearch(): void {
  for (const form of document.querySelectorAll<HTMLFormElement>(
    "[data-live-search]",
  ))
    setupLiveSearch(form);
}

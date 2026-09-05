// Administrator group-member picker behavior.

import { showNotice } from "../../core/dialogs.ts";
import { errorMessage, responseProblem } from "../../core/http.ts";

interface GroupMember {
  id: number;
  username: string;
  email?: string;
  display_name?: string;
}

function isGroupMember(value: unknown): value is GroupMember {
  if (typeof value !== "object" || value === null) return false;

  const member = value as Partial<GroupMember>;

  return typeof member.id === "number" && typeof member.username === "string";
}

function groupMembers(value: unknown): GroupMember[] {
  return Array.isArray(value) ? value.filter(isGroupMember) : [];
}

// Wires group member picker behavior.
export function setupGroupMemberPicker(card: HTMLElement): void {
  const input = card.querySelector<HTMLInputElement>(
    "[data-group-person-input]",
  );
  const results = card.querySelector<HTMLElement>(
    "[data-group-person-results]",
  );
  const memberList = card.querySelector<HTMLElement>(
    "[data-group-member-list]",
  );
  const count = card.querySelector<HTMLElement>("[data-group-member-count]");
  const groupID = card.dataset.groupId;
  const membersURL = card.dataset.membersUrl;
  const usersURL = card.dataset.usersUrl;
  if (!input || !results || !memberList || !groupID || !membersURL || !usersURL)
    return;

  const groupMembersURL = membersURL;
  const userSearchURL = usersURL;
  const personInput = input;
  const resultList = results;
  const renderedMembers = memberList;
  let members: GroupMember[] = [];
  let timer: ReturnType<typeof setTimeout> | undefined;
  let controller: AbortController | null = null;

  // Returns IDs for the currently rendered group members.
  function memberIDs(): Set<number> {
    return new Set(members.map((member) => member.id));
  }

  // Renders members.
  function renderMembers(): void {
    renderedMembers.replaceChildren();

    if (!members.length) {
      const empty = document.createElement("p");

      empty.className = "muted";
      empty.textContent = "No members yet.";
      renderedMembers.append(empty);
    }
    for (const member of members) {
      const chip = document.createElement("div");

      chip.className = "group-member-chip";

      const label = document.createElement("span");
      const name = document.createElement("strong");

      name.textContent = member.display_name || member.username;

      const detail = document.createElement("span");

      detail.textContent = member.email || `@${member.username}`;
      label.append(name, detail);

      const remove = document.createElement("button");

      remove.type = "button";
      remove.title = `Remove ${name.textContent}`;
      remove.setAttribute("aria-label", remove.title);
      remove.textContent = "×";
      remove.addEventListener("click", async () => {
        remove.disabled = true;
        try {
          const response = await fetch(`${groupMembersURL}/${member.id}`, {
            method: "DELETE",
            headers: { Accept: "application/json" },
          });
          if (!response.ok) throw await responseProblem(response);

          members = members.filter((item) => item.id !== member.id);
          renderMembers();
        } catch (error) {
          console.error("group member removal failed", error);
          await showNotice(
            errorMessage(error) || "Person could not be removed.",
            {
              title: "Member update failed",
            },
          );
          remove.disabled = false;
        }
      });
      chip.append(label, remove);
      renderedMembers.append(chip);
    }
    if (count) count.textContent = String(members.length);
  }

  // Loads members.
  async function loadMembers(): Promise<void> {
    try {
      const response = await fetch(groupMembersURL, {
        headers: { Accept: "application/json" },
      });
      if (!response.ok) throw await responseProblem(response);

      members = groupMembers(await response.json());
      renderMembers();
    } catch (error) {
      console.error("group members failed", error);
      renderedMembers.innerHTML =
        '<p class="revision-dialog-error">Members could not be loaded.</p>';
    }
  }

  // Renders results.
  function renderResults(users: GroupMember[]): void {
    resultList.replaceChildren();

    const existing = memberIDs();
    const available = users.filter((user) => !existing.has(user.id));

    if (!available.length) {
      const empty = document.createElement("div");

      empty.className = "muted live-person-result-empty";
      empty.textContent = "No matching people to add.";
      resultList.append(empty);
    }
    for (const user of available) {
      const option = document.createElement("button");

      option.type = "button";
      option.className = "live-person-result";
      option.setAttribute("role", "option");

      const avatar = document.createElement("span");

      avatar.className = "avatar";
      avatar.textContent = (user.display_name || user.username || "?")
        .slice(0, 1)
        .toUpperCase();

      const label = document.createElement("span");
      const name = document.createElement("strong");

      name.textContent = user.display_name || user.username;

      const detail = document.createElement("span");

      detail.textContent = user.email || `@${user.username}`;
      label.append(name, detail);
      option.append(avatar, label);
      option.addEventListener("click", async () => {
        option.disabled = true;
        try {
          const response = await fetch(groupMembersURL, {
            method: "POST",
            headers: {
              Accept: "application/json",
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ user_id: user.id }),
          });
          const payload: unknown = await response.json().catch(() => ({}));
          if (!response.ok) throw await responseProblem(response, payload);
          if (!isGroupMember(payload))
            throw new Error("Invalid member response.");

          if (!members.some((member) => member.id === payload.id))
            members.push(payload);

          renderMembers();
          personInput.value = "";
          resultList.hidden = true;
        } catch (error) {
          console.error("group member addition failed", error);
          await showNotice(
            errorMessage(error) || "Person could not be added.",
            {
              title: "Member update failed",
            },
          );
          option.disabled = false;
        }
      });
      resultList.append(option);
    }

    resultList.hidden = false;
  }

  // Searches people.
  async function searchPeople(): Promise<void> {
    const query = personInput.value.trim();
    if ([...query].length < 2) {
      controller?.abort();
      resultList.hidden = true;
      resultList.replaceChildren();
      return;
    }

    controller?.abort();
    controller = new AbortController();

    try {
      const response = await fetch(
        `${userSearchURL}?q=${encodeURIComponent(query)}`,
        {
          headers: { Accept: "application/json" },
          signal: controller.signal,
        },
      );
      if (!response.ok) throw await responseProblem(response);

      renderResults(groupMembers(await response.json()));
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") return;

      console.error("person search failed", error);
      resultList.hidden = true;
    }
  }

  personInput.addEventListener("input", () => {
    if (timer !== undefined) clearTimeout(timer);
    timer = setTimeout(() => void searchPeople(), 140);
  });
  personInput.addEventListener("blur", () =>
    setTimeout(() => {
      resultList.hidden = true;
    }, 120),
  );
  personInput.addEventListener("focus", () => {
    if (
      [...personInput.value.trim()].length >= 2 &&
      resultList.childElementCount
    )
      resultList.hidden = false;
  });

  void loadMembers();
}

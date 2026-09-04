import test from "node:test";
import assert from "node:assert/strict";

import {
  editorWordStats,
  slugifyEditorPath,
} from "../../web/src/ts/features/editor/experience.ts";
import { replaceAllPlainText } from "../../web/src/ts/features/editor/search.ts";
import { editorModeCopy } from "../../web/src/ts/features/editor/preview.ts";

test("editor path preview follows Lore slug rules", () => {
  assert.equal(slugifyEditorPath("Postgres Restore"), "postgres-restore");
  assert.equal(
    slugifyEditorPath("infrastructure/Postgres Restore"),
    "infrastructure/postgres-restore",
  );
  assert.equal(slugifyEditorPath("  API & Database  "), "api-database");
});

test("editor word statistics handle empty and multiline Markdown", () => {
  assert.deepEqual(editorWordStats(""), {
    words: 0,
    characters: 0,
    lines: 1,
  });
  assert.deepEqual(editorWordStats("# Hello\n\nLore wiki"), {
    words: 4,
    characters: 18,
    lines: 3,
  });
});

test("replace all supports case insensitive plain-text replacement", () => {
  assert.deepEqual(replaceAllPlainText("Lore lore LORE", "lore", "Wiki"), {
    value: "Wiki Wiki Wiki",
    count: 3,
  });
});

test("replace all can match case", () => {
  assert.deepEqual(
    replaceAllPlainText("Lore lore LORE", "Lore", "Wiki", true),
    { value: "Wiki lore LORE", count: 1 },
  );
});

test("editor mode copy matches the visible workspace", () => {
  assert.deepEqual(editorModeCopy("write"), {
    title: "Markdown",
    description: "Markdown stays the source of truth.",
  });
  assert.deepEqual(editorModeCopy("split"), {
    title: "Markdown & preview",
    description: "Edit Markdown with a live rendered preview.",
  });
  assert.deepEqual(editorModeCopy("preview"), {
    title: "Preview",
    description: "Rendered page preview.",
  });
});

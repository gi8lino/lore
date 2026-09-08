import test from "node:test";
import assert from "node:assert/strict";

import {
  editorWordStats,
  immediateEditorPathOptions,
  resolvedGuidedEditorPath,
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

test("guided editor paths combine the selected location and title", () => {
  assert.equal(
    resolvedGuidedEditorPath("Postgres Restore", "runbooks/database"),
    "runbooks/database/postgres-restore",
  );
  assert.equal(resolvedGuidedEditorPath("Top Level", ""), "top-level");
});

test("guided editor paths keep an existing final segment while moving", () => {
  assert.equal(
    resolvedGuidedEditorPath("Renamed title", "runbooks", "database-restore"),
    "runbooks/database-restore",
  );
});

test("guided locations suggest matching children one level at a time", () => {
  const options = [
    { slug: "applications", label: "Applications" },
    { slug: "platforms", label: "Platforms" },
    { slug: "platforms/containers", label: "Platforms / Containers" },
    { slug: "platforms/kubernetes", label: "Platforms / Kubernetes" },
    {
      slug: "platforms/kubernetes/tools",
      label: "Platforms / Kubernetes / Tools",
    },
  ];

  assert.deepEqual(immediateEditorPathOptions(options, "", "pla"), [
    { slug: "platforms", label: "Platforms" },
  ]);
  assert.deepEqual(immediateEditorPathOptions(options, "platforms", "k"), [
    { slug: "platforms/kubernetes", label: "Platforms / Kubernetes" },
  ]);
  assert.deepEqual(
    immediateEditorPathOptions(options, "platforms", "tools"),
    [],
  );
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

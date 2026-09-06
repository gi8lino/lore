// Runtime-checked metadata returned by GET /api/editor/catalog.

import { isRecord, isStringRecord, requireArrayOf } from "../../core/guards.ts";

export interface CatalogPage {
  slug: string;
  title: string;
}

export interface CatalogSnippet {
  kind: string;
  name: string;
  description?: string;
}

export interface EditorCatalog {
  pages: CatalogPage[];
  aliases: Record<string, string>;
  snippets: CatalogSnippet[];
}

function isCatalogPage(value: unknown): value is CatalogPage {
  return (
    isRecord(value) &&
    typeof value.slug === "string" &&
    typeof value.title === "string"
  );
}

function isCatalogSnippet(value: unknown): value is CatalogSnippet {
  return (
    isRecord(value) &&
    typeof value.kind === "string" &&
    typeof value.name === "string" &&
    (value.description === undefined || typeof value.description === "string")
  );
}

export function parseEditorCatalog(value: unknown): EditorCatalog {
  if (!isRecord(value) || !isStringRecord(value.aliases)) {
    throw new Error("Invalid editor catalog response.");
  }
  return {
    pages: requireArrayOf(value.pages, isCatalogPage, "editor catalog pages"),
    snippets: requireArrayOf(
      value.snippets,
      isCatalogSnippet,
      "editor catalog snippets",
    ),
    aliases: value.aliases,
  };
}

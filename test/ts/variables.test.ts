import assert from "node:assert/strict";
import test from "node:test";
import { variableOverrides } from "../../web/src/ts/features/variables.ts";
import { parseExportPreview } from "../../web/src/ts/features/export-preview.ts";

test("export variables omit unchanged values", () => {
  assert.deepEqual(
    variableOverrides([
      { name: "environment", saved: "production", value: "production" },
    ]),
    {},
  );
});

test("export variables keep an explicitly empty value", () => {
  assert.deepEqual(
    variableOverrides([
      { name: "environment", saved: "production", value: "" },
    ]),
    { environment: "" },
  );
});

test("export variables preserve whitespace", () => {
  assert.deepEqual(
    variableOverrides([
      { name: "environment", saved: "production", value: " staging\n " },
    ]),
    { environment: " staging\n " },
  );
});

test("export variables keep prototype-like names as own JSON properties", () => {
  const values = variableOverrides([
    { name: "__proto__", saved: "production", value: "staging" },
    { name: "constructor", saved: "old", value: "new" },
  ]);
  assert.equal(Object.getPrototypeOf(values), Object.prototype);
  assert.equal(
    JSON.stringify(values),
    '{"__proto__":"staging","constructor":"new"}',
  );
});

test("export variables do not modify their inputs", () => {
  const fields = [
    { name: "environment", saved: "production", value: "staging" },
  ];
  variableOverrides(fields);
  assert.deepEqual(fields, [
    { name: "environment", saved: "production", value: "staging" },
  ]);
});

test("export variables are independent between requests", () => {
  const first = variableOverrides([
    { name: "environment", saved: "production", value: "staging" },
  ]);
  const second = variableOverrides([
    { name: "environment", saved: "production", value: "production" },
  ]);
  assert.deepEqual(first, { environment: "staging" });
  assert.deepEqual(second, {});
});

test("export preview accepts a complete HTML document", () => {
  assert.equal(
    parseExportPreview({ document: "<!doctype html><p>staging</p>" }),
    "<!doctype html><p>staging</p>",
  );
});

test("export preview rejects a missing document", () => {
  let error: unknown;
  try {
    parseExportPreview({ html: "<p>wrong contract</p>" });
  } catch (caught) {
    error = caught;
  }
  assert.ok(error instanceof Error);
  assert.equal(error.message, "Invalid export preview response.");
});

test("export preview rejects an empty document", () => {
  let error: unknown;
  try {
    parseExportPreview({ document: "  " });
  } catch (caught) {
    error = caught;
  }
  assert.ok(error instanceof Error);
});

test("export preview rejects a non-string document", () => {
  let error: unknown;
  try {
    parseExportPreview({ document: {} });
  } catch (caught) {
    error = caught;
  }
  assert.ok(error instanceof Error);
});

import assert from "node:assert/strict";
import test from "node:test";

import { requiredElement } from "../../web/src/ts/core/dom.ts";

void test("requiredElement returns the matching template element", () => {
  const expected = {} as Element;
  const root = {
    querySelector: () => expected,
  } as unknown as ParentNode;

  assert.equal(requiredElement(root, "[data-required]"), expected);
});

void test("requiredElement fails loudly when required markup is missing", () => {
  const root = {
    querySelector: () => null,
  } as unknown as ParentNode;
  let caught: unknown;

  try {
    requiredElement(root, "[data-required]");
  } catch (error) {
    caught = error;
  }

  assert.ok(caught instanceof Error);
  assert.equal(caught.message, "Missing required element: [data-required]");
});

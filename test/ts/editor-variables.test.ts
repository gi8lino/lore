import test from "node:test";
import assert from "node:assert/strict";
import {
  variableReplacement,
  variableTrigger,
} from "../../web/src/ts/features/editor/variables.ts";

test("variableTrigger opens on an empty or filtered variable shortcut", () => {
  assert.deepEqual(variableTrigger("Deploy to {{", 12), {
    start: 10,
    query: "",
  });
  assert.deepEqual(variableTrigger("Deploy to {{env", 15), {
    start: 10,
    query: "env",
  });
});

test("variableTrigger ignores explicit macros and completed braces", () => {
  assert.equal(variableTrigger("{{var:environment", 17), null);
  assert.equal(variableTrigger("{{environment}}", 15), null);
});

test("variableTrigger ignores fenced code but works in inline code", () => {
  assert.equal(variableTrigger("```text\n{{env", 13), null);
  assert.deepEqual(variableTrigger("Use `{{env", 10), {
    start: 5,
    query: "env",
  });
});

test("variableReplacement inserts the canonical Lore macro", () => {
  assert.equal(variableReplacement("environment"), "{{var:environment}}");
});

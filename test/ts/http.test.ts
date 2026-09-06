import assert from "node:assert/strict";
import test from "node:test";

import { requestJSON, responseProblem } from "../../web/src/ts/core/http.ts";

void test("responseProblem validates problem payload fields", async () => {
  const response = new Response(JSON.stringify({ error: "fallback" }), {
    status: 400,
    headers: { "Content-Type": "application/json" },
  });

  const valid = await responseProblem(response, {
    error: "Invalid request",
    problems: { display_name: "Required" },
  });
  assert.equal(valid.message, "Invalid request\n\ndisplay name: Required");

  const invalid = await responseProblem(response, {
    error: false,
    problems: 42,
  });
  assert.equal(invalid.message, "HTTP 400");
});

void test("requestJSON returns unknown JSON and supplies an Accept header", async () => {
  const originalFetch = globalThis.fetch;
  let receivedAccept = "";

  globalThis.fetch = async (_input, init) => {
    receivedAccept = new Headers(init?.headers).get("Accept") ?? "";
    return new Response(JSON.stringify({ value: 7 }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    const payload = await requestJSON("https://example.test/api");

    assert.deepEqual(payload, { value: 7 });
    assert.equal(receivedAccept, "application/json");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

void test("requestJSON surfaces structured JSON errors", async () => {
  const originalFetch = globalThis.fetch;

  globalThis.fetch = async () =>
    new Response(
      JSON.stringify({
        error: "Invalid request",
        problems: { title: "Required" },
      }),
      {
        status: 422,
        headers: { "Content-Type": "application/json" },
      },
    );

  try {
    let caught: unknown;

    try {
      await requestJSON("https://example.test/api");
    } catch (error) {
      caught = error;
    }

    assert.ok(caught instanceof Error);
    assert.equal(caught.message, "Invalid request\n\ntitle: Required");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

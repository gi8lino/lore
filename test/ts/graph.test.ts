import test from "node:test";
import assert from "node:assert/strict";

import { graphNeighborhood } from "../../web/src/ts/features/graph.ts";

const graph = {
  nodes: [
    { slug: "a", title: "Alpha" },
    { slug: "b", title: "Beta" },
    { slug: "c", title: "Gamma" },
    { slug: "d", title: "Database" },
  ],
  edges: [
    { source: "a", target: "b" },
    { source: "b", target: "c" },
    { source: "d", target: "c" },
  ],
};

test("graphNeighborhood returns the complete graph without a filter", () => {
  assert.deepEqual(graphNeighborhood(graph, "", ""), graph);
});

test("graphNeighborhood returns a focused node and its direct neighbors", () => {
  const result = graphNeighborhood(graph, "b");

  assert.deepEqual(result.nodes.map((node) => node.slug).sort(), [
    "a",
    "b",
    "c",
  ]);
  assert.equal(result.edges.length, 2);
});

test("graphNeighborhood expands matching search results with their relationships", () => {
  const result = graphNeighborhood(graph, "", "database");

  assert.deepEqual(result.nodes.map((node) => node.slug).sort(), ["c", "d"]);
  assert.deepEqual(result.edges, [{ source: "d", target: "c" }]);
});

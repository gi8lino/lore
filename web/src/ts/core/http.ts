// Small HTTP response helpers for browser requests.

import { isRecord, isStringRecord } from "./guards.ts";

export interface ProblemPayload {
  error?: string;
  problems?: Record<string, string>;
}

function isProblemPayload(value: unknown): value is ProblemPayload {
  if (!isRecord(value)) return false;

  if (value.error !== undefined && typeof value.error !== "string") {
    return false;
  }
  if (value.problems !== undefined && !isStringRecord(value.problems)) {
    return false;
  }

  return true;
}

export async function responseProblem(
  response: Response,
  payload?: unknown,
): Promise<Error> {
  let candidate = payload;

  if (candidate === undefined) {
    candidate = await response
      .clone()
      .json()
      .catch(() => undefined);
  }

  const problem = isProblemPayload(candidate) ? candidate : {};
  const details = Object.entries(problem.problems ?? {})
    .filter(([, message]) => message.trim() !== "")
    .map(([field, message]) => `${field.replaceAll("_", " ")}: ${message}`);
  const message = problem.error?.trim() || `HTTP ${response.status}`;

  return new Error(
    details.length ? `${message}\n\n${details.join("\n")}` : message,
  );
}

// Performs a JSON request without trusting the decoded response shape.
export async function requestJSON(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<unknown> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) headers.set("Accept", "application/json");

  const response = await fetch(input, { ...init, headers });
  const payload: unknown = await response.json().catch(() => undefined);

  if (!response.ok) throw await responseProblem(response, payload);

  return payload;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

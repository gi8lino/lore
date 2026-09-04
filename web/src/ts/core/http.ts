// Small HTTP response helpers for browser requests.

export interface ProblemPayload {
  error?: string;
  problems?: Record<string, string>;
}

function isProblemPayload(value: unknown): value is ProblemPayload {
  return typeof value === "object" && value !== null;
}

export async function responseProblem(
  response: Response,
  payload: unknown = null,
): Promise<Error> {
  let body = payload;
  if (!isProblemPayload(body)) {
    body = await response.json().catch(() => ({}));
  }
  const problem = isProblemPayload(body) ? body : {};
  const details = Object.entries(problem.problems ?? {})
    .filter(([, message]) => message)
    .map(([field, message]) => `${field.replaceAll("_", " ")}: ${message}`);
  const message = problem.error || `HTTP ${response.status}`;
  return new Error(
    details.length ? `${message}\n\n${details.join("\n")}` : message,
  );
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

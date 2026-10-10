import "server-only";

import { z } from "zod";

import type { APIError, Result } from "./result";

export interface RequestOptions<T> {
  url: string;
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  /** Sent as JSON. Translate it to the wire shape (`snake_case`) first. */
  body?: unknown;
  /** Parses the success body; an empty body parses as `undefined`. */
  schema: z.ZodType<T>;
}

const unavailable: APIError = {
  code: "UNAVAILABLE",
  message: "service unavailable",
  retryable: true,
  status: 503,
};

const unexpected: APIError = {
  code: "UNEXPECTED",
  message: "unexpected response from service",
  retryable: false,
  status: 502,
};

/** go-kit's `httpserver/util.APIError` as written to the response body. */
const apiErrorBodySchema = z.object({
  code: z.string(),
  message: z.string(),
  retryable: z.boolean(),
  details: z.object({ reasons: z.array(z.string()).optional() }).optional(),
});

type Body = { kind: "json"; value: unknown } | { kind: "invalid" };

function parseBody(text: string): Body {
  if (text === "") return { kind: "json", value: undefined };
  try {
    return { kind: "json", value: JSON.parse(text) };
  } catch {
    return { kind: "invalid" };
  }
}

/**
 * Calls a backend service and never throws: a network failure becomes
 * `UNAVAILABLE`, a body outside the schema becomes `UNEXPECTED`, and an error
 * status returns the service's `APIError`.
 */
export async function request<T>({
  url,
  method = "GET",
  body,
  schema,
}: RequestOptions<T>): Promise<Result<T>> {
  let status: number;
  let text: string;
  try {
    const response = await fetch(url, {
      method,
      cache: "no-store",
      ...(body === undefined
        ? {}
        : {
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          }),
    });
    status = response.status;
    text = await response.text();
  } catch {
    return { kind: "error", error: unavailable };
  }

  const parsedBody = parseBody(text);
  if (parsedBody.kind === "invalid") {
    return { kind: "error", error: unexpected };
  }

  if (status >= 400) {
    const apiError = apiErrorBodySchema.safeParse(parsedBody.value);
    if (!apiError.success) return { kind: "error", error: unexpected };
    return { kind: "error", error: { ...apiError.data, status } };
  }

  const data = schema.safeParse(parsedBody.value);
  if (!data.success) return { kind: "error", error: unexpected };
  return { kind: "ok", data: data.data };
}

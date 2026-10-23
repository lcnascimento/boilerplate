import "server-only";

import { z } from "zod";

import { unavailableError, unexpectedError } from "./errors";

import type { Result } from "./result";

export interface RequestOptions<T> {
  url: string;
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  token?: string;
  schema: z.ZodType<T>;
}

export type GetToken = () => Promise<string>;

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

export async function request<T>({
  url,
  method = "GET",
  body,
  token,
  schema,
}: RequestOptions<T>): Promise<Result<T>> {
  const headers = new Headers();
  if (token !== undefined) headers.set("Authorization", `Bearer ${token}`);
  if (body !== undefined) headers.set("Content-Type", "application/json");

  let status: number;
  let text: string;
  try {
    const response = await fetch(url, {
      method,
      cache: "no-store",
      headers,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    status = response.status;
    text = await response.text();
  } catch {
    return { kind: "error", error: unavailableError };
  }

  const parsedBody = parseBody(text);
  if (parsedBody.kind === "invalid") {
    return { kind: "error", error: unexpectedError };
  }

  if (status >= 400) {
    const apiError = apiErrorBodySchema.safeParse(parsedBody.value);
    if (!apiError.success) return { kind: "error", error: unexpectedError };
    return { kind: "error", error: { ...apiError.data, status } };
  }

  const data = schema.safeParse(parsedBody.value);
  if (!data.success) return { kind: "error", error: unexpectedError };
  return { kind: "ok", data: data.data };
}

import "server-only";

import { APIError } from "better-auth/api";
import { redirect } from "next/navigation";

import { userIdSchema } from "@/shared/auth/actor-ids";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { getRequestSession } from "@/shared/auth/session";
import { logger } from "@/shared/logging";

import type { Actor } from "@/shared/auth/actor";
import type { AuditedActor } from "@/shared/auth/audit-connection";
import type { z } from "zod";

export const invalidInput = { kind: "error", code: "invalidInput" } as const;
export const forbidden = { kind: "error", code: "forbidden" } as const;
export const unexpected = { kind: "error", code: "unexpected" } as const;

type InvalidInput = typeof invalidInput;

export function serverAction<Schema extends z.ZodType, Result>(
  schema: Schema,
  handler: (input: z.output<Schema>) => Promise<Result>
): (input: unknown) => Promise<Result | InvalidInput>;
export function serverAction<Schema extends z.ZodType, Result>(
  schema: Schema,
  handler: (input: z.output<Schema>) => Promise<Result>,
  rejectInput: (error: z.ZodError) => Result
): (input: unknown) => Promise<Result>;
export function serverAction<Schema extends z.ZodType, Result>(
  schema: Schema,
  handler: (input: z.output<Schema>) => Promise<Result>,
  rejectInput: (error: z.ZodError) => Result | InvalidInput = () => invalidInput
): (input: unknown) => Promise<Result | InvalidInput> {
  return async (input) => {
    const parsed = schema.safeParse(input);
    if (!parsed.success) return rejectInput(parsed.error);
    return handler(parsed.data);
  };
}

export async function requireSession() {
  const session = await getRequestSession();
  if (session === null) redirect(SIGN_IN_HREF);
  return session;
}

export async function requireActor(): Promise<Actor | null> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  return requestActor.kind === "actor" ? requestActor.actor : null;
}

export async function requireAuditedActor(): Promise<AuditedActor> {
  const { user } = await requireSession();
  const requestActor = await getRequestActor();
  return {
    userId: userIdSchema.parse(user.id),
    organizationId:
      requestActor.kind === "actor" ? requestActor.actor.organizationId : null,
  };
}

export function logFailure(operation: string, error: unknown) {
  logger.error({ err: error, operation }, "server action failed");
}

export async function runOperation<Code extends string>(
  operation: string,
  refusals: Readonly<Record<string, Code>>,
  run: () => Promise<unknown>
): Promise<
  { kind: "ok" } | { kind: "error"; code: Code | typeof unexpected.code }
> {
  try {
    await run();
    return { kind: "ok" };
  } catch (error) {
    const code =
      error instanceof APIError && typeof error.body?.code === "string"
        ? refusals[error.body.code]
        : undefined;
    if (code !== undefined) return { kind: "error", code };

    logFailure(operation, error);
    return unexpected;
  }
}

"use server";

import { headers } from "next/headers";
import { redirect } from "next/navigation";

import {
  deleteAccountInputSchema,
  userNameInputSchema,
  type AccountDeletionResult,
  type RenameUserResult,
} from "@/shared/auth/account";
import {
  accountDeletionBlockers,
  deleteAccount as deleteAccountOf,
} from "@/shared/auth/account-deletion";
import { userIdSchema } from "@/shared/auth/actor-ids";
import { getIam } from "@/shared/auth/auth";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { getRequestSession } from "@/shared/auth/session";
import { logger } from "@/shared/logging";

import type { Organization } from "@/shared/auth/organizations";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

const invalidInput = { kind: "error", code: "invalidInput" } as const;
const unexpected = { kind: "error", code: "unexpected" } as const;

async function requireSession() {
  const session = await getRequestSession();
  if (session === null) redirect(SIGN_IN_HREF);
  return session;
}

function logFailure(operation: string, error: unknown) {
  logger.error({ err: error, operation }, "account operation failed");
}

export async function getAccountSettings(): Promise<{
  name: string;
  deletionBlockers: Organization[];
}> {
  const { user } = await requireSession();
  return {
    name: user.name,
    deletionBlockers: await accountDeletionBlockers(
      userIdSchema.parse(user.id)
    ),
  };
}

export async function renameUser(input: unknown): Promise<RenameUserResult> {
  const parsed = userNameInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  await requireSession();

  try {
    await getIam().auth.api.updateUser({
      headers: requestHeaders,
      body: { name: parsed.data.name },
    });
    return { kind: "ok" };
  } catch (error) {
    logFailure("account.rename", error);
    return unexpected;
  }
}

export async function deleteAccount(
  input: unknown
): Promise<AccountDeletionResult> {
  const parsed = deleteAccountInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  const { user } = await requireSession();

  let result: AccountDeletionResult;
  try {
    result = await deleteAccountOf(
      userIdSchema.parse(user.id),
      parsed.data.password
    );
  } catch (error) {
    logFailure("account.delete", error);
    return unexpected;
  }

  if (result.kind === "ok") {
    try {
      await getIam().auth.api.signOut({ headers: requestHeaders });
    } catch (error) {
      logFailure("account.delete.signOut", error);
    }
  }
  return result;
}

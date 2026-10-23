"use server";

import { headers } from "next/headers";

import {
  deleteAccountInputSchema,
  userNameInputSchema,
  type AccountDeletionResult,
  type RenameUserResult,
} from "@/shared/auth/account";
import {
  accountDeletionBlockers,
  deleteAccountFor,
} from "@/shared/auth/account-deletion";
import { userIdSchema } from "@/shared/auth/actor-ids";
import { getIam } from "@/shared/auth/auth";

import {
  logFailure,
  requireAuditedActor,
  requireSession,
  serverAction,
  unexpected,
} from "./server-action";

import type { Organization } from "@/shared/auth/organizations";

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

export const renameUser = serverAction(
  userNameInputSchema,
  async ({ name }): Promise<RenameUserResult> => {
    const requestHeaders = await headers();
    await requireSession();

    try {
      await getIam().auth.api.updateUser({
        headers: requestHeaders,
        body: { name },
      });
      return { kind: "ok" };
    } catch (error) {
      logFailure("account.rename", error);
      return unexpected;
    }
  }
);

export const deleteAccount = serverAction(
  deleteAccountInputSchema,
  async ({ password }): Promise<AccountDeletionResult> => {
    const requestHeaders = await headers();
    const deletingUser = await requireAuditedActor();

    let result: AccountDeletionResult;
    try {
      result = await deleteAccountFor(
        deletingUser,
        new Headers(requestHeaders),
        password
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
);

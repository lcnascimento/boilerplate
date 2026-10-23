import { z } from "zod";

import { PASSWORD_LENGTH, USER_NAME_MAX_LENGTH } from "./policy";

import type { Organization } from "./organizations";

export const userNameInputSchema = z.object({
  name: z.string().trim().min(1).max(USER_NAME_MAX_LENGTH),
});

export const changePasswordInputSchema = z.object({
  currentPassword: z.string().min(1),
  newPassword: z.string().min(PASSWORD_LENGTH.min).max(PASSWORD_LENGTH.max),
});

export const deleteAccountInputSchema = z.object({
  password: z.string().min(1),
});

export type RenameUserResult =
  { kind: "ok" } | { kind: "error"; code: "invalidInput" | "unexpected" };

export type AccountDeletionResult =
  | { kind: "ok" }
  | { kind: "error"; code: "invalidInput" | "invalidPassword" | "unexpected" }
  | {
      kind: "error";
      code: "ownsSharedOrganizations";
      organizations: readonly Organization[];
    };

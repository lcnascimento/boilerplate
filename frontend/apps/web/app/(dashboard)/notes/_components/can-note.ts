import { roles } from "@/shared/auth/authz.gen";

import type { MemberRole } from "@/shared/auth/members";

export function canNote(
  role: MemberRole | null,
  action: "create" | "delete"
): boolean {
  return role !== null && roles[role].authorize({ note: [action] }).success;
}

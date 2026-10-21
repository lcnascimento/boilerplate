import { roles } from "@/shared/auth/roles.gen";

import type { MemberRole } from "@/shared/auth/members";

// Hides what the Role cannot do; the notes service is the one that decides.
export function canNote(
  role: MemberRole | null,
  action: "create" | "delete"
): boolean {
  return role !== null && roles[role].authorize({ note: [action] }).success;
}

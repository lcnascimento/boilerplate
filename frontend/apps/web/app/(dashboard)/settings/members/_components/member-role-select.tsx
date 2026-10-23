"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@boilerplate/ui/components/select";

import { updateMemberRole } from "@/actions/members";
import {
  memberRoleSchema,
  type Member,
  type MemberRole,
} from "@/shared/auth/members";

interface MemberRoleSelectProps {
  member: Pick<Member, "id" | "name" | "role">;
  assignableRoles: MemberRole[];
}

export function MemberRoleSelect({
  member,
  assignableRoles,
}: MemberRoleSelectProps) {
  const t = useTranslations("members");
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  function handleChange(value: string) {
    const role = memberRoleSchema.parse(value);
    if (role === member.role) return;

    startTransition(async () => {
      const result = await updateMemberRole({ memberId: member.id, role });
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`));
        return;
      }
      toast.success(t("roleChanged"));
      router.refresh();
    });
  }

  return (
    <Select
      disabled={isPending}
      value={member.role}
      onValueChange={handleChange}
    >
      <SelectTrigger
        aria-busy={isPending}
        aria-label={t("changeRoleOf", { name: member.name })}
        size="sm"
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {assignableRoles.map((role) => (
          <SelectItem key={role} value={role}>
            {t(`roles.${role}`)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

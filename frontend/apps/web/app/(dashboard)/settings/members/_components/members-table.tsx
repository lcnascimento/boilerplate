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
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@boilerplate/ui/components/table";

import { updateMemberRole } from "@/actions/members";
import {
  memberRoleSchema,
  type Member,
  type MemberPolicy,
  type MemberRole,
  type MemberRow,
} from "@/shared/auth/members";

import { RemoveMemberDialog } from "./remove-member-dialog";

interface MembersTableProps {
  actorId: Member["id"];
  policy: MemberPolicy;
}

export function MembersTable({ actorId, policy }: MembersTableProps) {
  const t = useTranslations("members");

  return (
    <Table>
      <TableCaption className="sr-only">{t("tableCaption")}</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>{t("nameColumn")}</TableHead>
          <TableHead>{t("emailColumn")}</TableHead>
          <TableHead>{t("roleColumn")}</TableHead>
          <TableHead>
            <span className="sr-only">{t("actionsColumn")}</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {policy.rows.map((row) => (
          <MemberTableRow
            key={row.member.id}
            assignableRoles={policy.assignableRoles}
            isSelf={row.member.id === actorId}
            row={row}
          />
        ))}
      </TableBody>
    </Table>
  );
}

interface MemberTableRowProps {
  row: MemberRow;
  assignableRoles: MemberRole[];
  isSelf: boolean;
}

function MemberTableRow({
  row: { member, canChangeRole, canRemove },
  assignableRoles,
  isSelf,
}: MemberTableRowProps) {
  const t = useTranslations("members");

  return (
    <TableRow>
      <TableCell className="font-medium">
        {isSelf ? `${member.name} (${t("you")})` : member.name}
      </TableCell>
      <TableCell>{member.email}</TableCell>
      <TableCell>
        {canChangeRole ? (
          <MemberRoleSelect assignableRoles={assignableRoles} member={member} />
        ) : (
          t(`roles.${member.role}`)
        )}
      </TableCell>
      <TableCell className="text-right">
        {canRemove ? <RemoveMemberDialog member={member} /> : null}
      </TableCell>
    </TableRow>
  );
}

interface MemberRoleSelectProps {
  member: Member;
  assignableRoles: MemberRole[];
}

function MemberRoleSelect({ member, assignableRoles }: MemberRoleSelectProps) {
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

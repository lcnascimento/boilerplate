import { useTranslations } from "next-intl";

import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@boilerplate/ui/components/table";

import { MemberRoleSelect } from "./member-role-select";
import { RemoveMemberDialog } from "./remove-member-dialog";

import type {
  Member,
  MemberPolicy,
  MemberRole,
  MemberRow,
} from "@/shared/auth/members";

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
  const { id, name, role } = member;

  return (
    <TableRow>
      <TableCell className="font-medium">
        {isSelf ? t("nameWithYou", { name }) : name}
      </TableCell>
      <TableCell>{member.email}</TableCell>
      <TableCell>
        {canChangeRole ? (
          <MemberRoleSelect
            assignableRoles={assignableRoles}
            member={{ id, name, role }}
          />
        ) : (
          t(`roles.${role}`)
        )}
      </TableCell>
      <TableCell className="text-right">
        {canRemove ? <RemoveMemberDialog member={{ id, name }} /> : null}
      </TableCell>
    </TableRow>
  );
}

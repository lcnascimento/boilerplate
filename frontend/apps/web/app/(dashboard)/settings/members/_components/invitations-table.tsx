import { useFormatter, useTranslations } from "next-intl";

import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@boilerplate/ui/components/table";

import { CancelInvitationButton } from "./cancel-invitation-button";

import type { SentInvitation } from "@/shared/auth/invitations";

interface InvitationsTableProps {
  invitations: readonly SentInvitation[];
  canCancel: boolean;
}

export function InvitationsTable({
  invitations,
  canCancel,
}: InvitationsTableProps) {
  const t = useTranslations("invitations");
  const tMembers = useTranslations("members");
  const format = useFormatter();

  if (invitations.length === 0) return null;

  return (
    <section aria-labelledby="pending-invitations" className="grid gap-2">
      <h2 className="text-lg font-semibold" id="pending-invitations">
        {t("pendingTitle")}
      </h2>
      <Table>
        <TableCaption className="sr-only">{t("tableCaption")}</TableCaption>
        <TableHeader>
          <TableRow>
            <TableHead>{t("emailColumn")}</TableHead>
            <TableHead>{t("roleColumn")}</TableHead>
            <TableHead>{t("expiresColumn")}</TableHead>
            <TableHead>
              <span className="sr-only">{t("actionsColumn")}</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {invitations.map(({ id, email, role, expiresAt }) => (
            <TableRow key={id}>
              <TableCell className="font-medium">{email}</TableCell>
              <TableCell>{tMembers(`roles.${role}`)}</TableCell>
              <TableCell>
                {format.dateTime(expiresAt, { dateStyle: "short" })}
              </TableCell>
              <TableCell className="text-right">
                {canCancel ? (
                  <CancelInvitationButton invitation={{ id, email }} />
                ) : null}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </section>
  );
}

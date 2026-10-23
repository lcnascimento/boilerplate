"use client";

import { useRouter } from "next/navigation";
import { useFormatter, useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";

import { acceptInvitation, rejectInvitation } from "@/actions/invitations";
import { ORGANIZATION_LIMIT } from "@/shared/auth/policy";
import { SIGNED_IN_HOME } from "@/shared/auth/routes";

import type {
  InvitationsResult,
  ReceivedInvitation,
} from "@/shared/auth/invitations";

interface ReceivedInvitationsProps {
  invitations: readonly ReceivedInvitation[];
}

export function ReceivedInvitations({ invitations }: ReceivedInvitationsProps) {
  const t = useTranslations("invitations");

  if (invitations.length === 0) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2>{t("receivedTitle")}</h2>
        </CardTitle>
        <CardDescription>{t("receivedDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="grid gap-4">
          {invitations.map((invitation) => (
            <ReceivedInvitationItem
              key={invitation.id}
              invitation={invitation}
            />
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function ReceivedInvitationItem({
  invitation: { id, organizationName, role, expiresAt },
}: {
  invitation: ReceivedInvitation;
}) {
  const t = useTranslations("invitations");
  const tMembers = useTranslations("members");
  const format = useFormatter();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  function respond(
    answer: (input: { invitationId: string }) => Promise<InvitationsResult>,
    onDone: () => void
  ) {
    startTransition(async () => {
      const result = await answer({ invitationId: id });
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`, { limit: ORGANIZATION_LIMIT }));
        return;
      }
      onDone();
    });
  }

  return (
    <li className="flex flex-wrap items-center justify-between gap-3">
      <div className="grid gap-0.5">
        <span className="font-medium">{organizationName}</span>
        <span className="text-sm text-muted-foreground">
          {t("receivedRole", {
            role: tMembers(`roles.${role}`),
            expiresAt: format.dateTime(expiresAt, { dateStyle: "short" }),
          })}
        </span>
      </div>
      <div className="flex gap-2">
        <Button
          aria-busy={isPending}
          aria-label={t("acceptInvitationTo", {
            organization: organizationName,
          })}
          disabled={isPending}
          size="sm"
          onClick={() =>
            respond(acceptInvitation, () => {
              router.replace(SIGNED_IN_HOME);
              router.refresh();
            })
          }
        >
          {t("accept")}
        </Button>
        <Button
          aria-label={t("rejectInvitationTo", {
            organization: organizationName,
          })}
          disabled={isPending}
          size="sm"
          variant="outline"
          onClick={() =>
            respond(rejectInvitation, () => {
              toast.success(t("rejected"));
              router.refresh();
            })
          }
        >
          {t("reject")}
        </Button>
      </div>
    </li>
  );
}

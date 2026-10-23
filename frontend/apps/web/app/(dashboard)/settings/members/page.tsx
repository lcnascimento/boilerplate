import { getTranslations } from "next-intl/server";

import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listSentInvitations } from "@/actions/invitations";
import { listMembers } from "@/actions/members";
import { memberPolicy } from "@/shared/auth/members";

import {
  InvitationsTable,
  InviteMemberDialog,
  MembersTable,
} from "./_components";

export default async function MembersPage() {
  const [{ actor, members }, invitations, t] = await Promise.all([
    listMembers(),
    listSentInvitations(),
    getTranslations("members"),
  ]);
  const policy = memberPolicy(actor, members);

  return (
    <>
      <PageHeader
        action={policy.canInvite ? <InviteMemberDialog /> : null}
        description={t("description")}
        title={t("title")}
      />
      <div className="grid gap-6">
        <MembersTable actorId={actor.id} policy={policy} />
        <InvitationsTable
          canCancel={policy.canCancelInvitations}
          invitations={invitations}
        />
      </div>
    </>
  );
}

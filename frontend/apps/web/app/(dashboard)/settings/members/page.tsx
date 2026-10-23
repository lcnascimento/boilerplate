import { getTranslations } from "next-intl/server";

import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listMembers } from "@/actions/members";
import { memberPolicy } from "@/shared/auth/members";

import { AddMemberDialog, MembersTable } from "./_components";

export default async function MembersPage() {
  const [{ actor, members }, t] = await Promise.all([
    listMembers(),
    getTranslations("members"),
  ]);
  const policy = memberPolicy(actor, members);

  return (
    <>
      <PageHeader
        action={
          policy.canAdd ? (
            <AddMemberDialog assignableRoles={policy.assignableRoles} />
          ) : null
        }
        description={t("description")}
        title={t("title")}
      />
      <MembersTable actorId={actor.id} policy={policy} />
    </>
  );
}

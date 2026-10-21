import { getTranslations } from "next-intl/server";

import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listMembers } from "@/actions/members";
import { memberPolicy } from "@/shared/auth/members";

import {
  AddMemberDialog,
  LeaveOrganization,
  MembersTable,
} from "./_components";

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
      <div className="grid gap-6">
        <MembersTable actorId={actor.id} policy={policy} />
        <LeaveOrganization leave={policy.leave} />
      </div>
    </>
  );
}

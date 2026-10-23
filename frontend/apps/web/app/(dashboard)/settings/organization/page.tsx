import { getTranslations } from "next-intl/server";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listMembers } from "@/actions/members";
import { getActiveOrganization } from "@/actions/organization";
import { ownershipPolicy } from "@/shared/auth/members";
import { organizationSettingsPolicy } from "@/shared/auth/organizations";

import {
  DeleteOrganization,
  LeaveOrganization,
  RenameOrganization,
  TransferOwnership,
} from "./_components";

export default async function OrganizationSettingsPage() {
  const [organization, { actor, members }, tNav, t] = await Promise.all([
    getActiveOrganization(),
    listMembers(),
    getTranslations("nav"),
    getTranslations("organizationSettings"),
  ]);
  const policy = organizationSettingsPolicy(actor.role);

  return (
    <>
      <PageHeader
        description={t("description")}
        title={tNav("organizationSettings")}
      />
      <div className="grid max-w-2xl gap-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("nameTitle")}</CardTitle>
            <CardDescription>{t("nameDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <RenameOrganization
              key={`${organization.id}:${organization.name}`}
              canRename={policy.canRename}
              name={organization.name}
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("transferTitle")}</CardTitle>
            <CardDescription>{t("transferDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <TransferOwnership policy={ownershipPolicy(actor, members)} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("leaveTitle")}</CardTitle>
            <CardDescription>{t("leaveDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <LeaveOrganization leave={policy.leave} />
          </CardContent>
        </Card>
        {policy.canDelete ? (
          <Card className="border-destructive/50">
            <CardHeader>
              <CardTitle>{t("deleteTitle")}</CardTitle>
              <CardDescription>{t("deleteDescription")}</CardDescription>
            </CardHeader>
            <CardContent>
              <DeleteOrganization name={organization.name} />
            </CardContent>
          </Card>
        ) : null}
      </div>
    </>
  );
}

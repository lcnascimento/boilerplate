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
import { ownershipPolicy } from "@/shared/auth/members";

import { TransferOwnership } from "./_components";

export default async function OrganizationSettingsPage() {
  const [{ actor, members }, tNav, t] = await Promise.all([
    listMembers(),
    getTranslations("nav"),
    getTranslations("organizationSettings"),
  ]);

  return (
    <>
      <PageHeader
        description={t("description")}
        title={tNav("organizationSettings")}
      />
      <div className="grid max-w-2xl gap-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("transferTitle")}</CardTitle>
            <CardDescription>{t("transferDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <TransferOwnership policy={ownershipPolicy(actor, members)} />
          </CardContent>
        </Card>
      </div>
    </>
  );
}

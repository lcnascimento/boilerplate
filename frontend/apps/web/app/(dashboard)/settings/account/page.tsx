import { getTranslations } from "next-intl/server";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { getAccountSettings } from "@/actions/account";

import { ChangePassword, DeleteAccount, RenameUser } from "./_components";

export default async function AccountSettingsPage() {
  const [{ name, deletionBlockers }, tNav, t] = await Promise.all([
    getAccountSettings(),
    getTranslations("nav"),
    getTranslations("accountSettings"),
  ]);

  return (
    <>
      <PageHeader
        description={t("description")}
        title={tNav("accountSettings")}
      />
      <div className="grid max-w-2xl gap-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("nameTitle")}</CardTitle>
            <CardDescription>{t("nameDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <RenameUser key={name} name={name} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("passwordTitle")}</CardTitle>
            <CardDescription>{t("passwordDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <ChangePassword />
          </CardContent>
        </Card>
        <Card className="border-destructive/50">
          <CardHeader>
            <CardTitle>{t("deleteTitle")}</CardTitle>
            <CardDescription>{t("deleteDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <DeleteAccount blockers={deletionBlockers} />
          </CardContent>
        </Card>
      </div>
    </>
  );
}

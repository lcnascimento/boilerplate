import { useTranslations } from "next-intl";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { LocalePreference, ThemePreference } from "./_components";

export default function PreferencesPage() {
  const tNav = useTranslations("nav");
  const t = useTranslations("profile");

  return (
    <>
      <PageHeader
        description={t("preferencesDescription")}
        title={tNav("preferences")}
      />
      <div className="grid max-w-2xl gap-4">
        <Card>
          <CardHeader>
            <CardTitle id="theme-title">{t("theme")}</CardTitle>
            <CardDescription>{t("themeDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <ThemePreference labelledBy="theme-title" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle id="language-title">{t("language")}</CardTitle>
            <CardDescription>{t("languageDescription")}</CardDescription>
          </CardHeader>
          <CardContent>
            <LocalePreference labelledBy="language-title" />
          </CardContent>
        </Card>
      </div>
    </>
  );
}

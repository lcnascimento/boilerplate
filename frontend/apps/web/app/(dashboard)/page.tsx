import { useTranslations } from "next-intl";

import { PageHeader } from "@boilerplate/ui/primitives/page-header";

export default function HomePage() {
  const t = useTranslations("nav");

  return (
    <PageHeader
      description={t("welcomeDescription")}
      title={t("welcomeTitle")}
    />
  );
}

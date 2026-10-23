import { Building2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

import { SignOutButton } from "./sign-out-button";

export function NoOrganization() {
  const t = useTranslations("organizations");

  return (
    <main className="flex min-h-svh items-center justify-center p-6">
      <EmptyState
        action={<SignOutButton />}
        className="w-full max-w-md"
        description={t("noOrganizationDescription")}
        icon={Building2}
        title={t("noOrganizationTitle")}
      />
    </main>
  );
}

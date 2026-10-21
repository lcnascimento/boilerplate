"use client";

import { Building2, LogOut } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";
import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

import { useSignOut } from "@/shared/hooks/use-sign-out";

export function NoOrganization() {
  const t = useTranslations("organizations");
  const tAuth = useTranslations("auth");
  const { signOut, isSigningOut } = useSignOut();

  return (
    <main className="flex min-h-svh items-center justify-center p-6">
      <EmptyState
        className="w-full max-w-md"
        description={t("noOrganizationDescription")}
        icon={Building2}
        title={t("noOrganizationTitle")}
        action={
          <Button
            aria-busy={isSigningOut}
            disabled={isSigningOut}
            onClick={signOut}
          >
            <LogOut aria-hidden />
            {tAuth("signOut")}
          </Button>
        }
      />
    </main>
  );
}

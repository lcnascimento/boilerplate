"use client";

import { LogOut } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";

import { useSignOut } from "@/shared/hooks/use-sign-out";

export function SignOutButton() {
  const t = useTranslations("auth");
  const { signOut, isSigningOut } = useSignOut();

  return (
    <Button aria-busy={isSigningOut} disabled={isSigningOut} onClick={signOut}>
      <LogOut aria-hidden />
      {t("signOut")}
    </Button>
  );
}

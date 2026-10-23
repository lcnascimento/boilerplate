"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import { authClient } from "@/shared/auth/client";
import { SIGN_IN_HREF } from "@/shared/auth/routes";

export function useSignOut() {
  const t = useTranslations("auth");
  const router = useRouter();
  const [isSigningOut, startSignOut] = useTransition();

  function signOut() {
    startSignOut(async () => {
      const { error } = await authClient.signOut();
      if (error) {
        toast.error(t("signOutFailed"));
        return;
      }
      router.replace(SIGN_IN_HREF);
      router.refresh();
    });
  }

  return { signOut, isSigningOut };
}

"use client";

import { useRouter } from "next/navigation";
import { useLocale } from "next-intl";
import { useTransition } from "react";

import { isLocale, type Locale } from "@boilerplate/i18n";

import { writeLocaleCookie } from "@/shared/cookies";

/**
 * The active locale and a setter that writes the locale cookie and re-renders
 * the server tree, so every text changes without a full reload.
 */
export function useLocaleSwitch(): {
  locale: Locale;
  setLocale: (value: string) => void;
  isPending: boolean;
} {
  const locale = useLocale();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  function setLocale(value: string) {
    if (!isLocale(value) || value === locale) return;
    writeLocaleCookie(value);
    startTransition(() => {
      router.refresh();
    });
  }

  return { locale, setLocale, isPending };
}

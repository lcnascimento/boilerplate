"use client";

import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";

import { localeConfig, locales } from "@boilerplate/i18n";
import {
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from "@boilerplate/ui/components/dropdown-menu";

import { useLocaleSwitch } from "@/shared/hooks/use-locale-switch";
import { parseTheme, themeOptions, themes } from "@/shared/theme";

/** Theme choices for a dropdown menu (header switcher and user menu). */
export function ThemeRadioItems() {
  const t = useTranslations("profile");
  const { theme, setTheme } = useTheme();

  return (
    <DropdownMenuRadioGroup value={parseTheme(theme)} onValueChange={setTheme}>
      {themes.map((value) => {
        const { labelKey, icon: Icon } = themeOptions[value];
        return (
          <DropdownMenuRadioItem key={value} value={value}>
            <Icon aria-hidden />
            {t(labelKey)}
          </DropdownMenuRadioItem>
        );
      })}
    </DropdownMenuRadioGroup>
  );
}

/** Locale choices for a dropdown menu (header switcher and user menu). */
export function LocaleRadioItems() {
  const { locale, setLocale } = useLocaleSwitch();

  return (
    <DropdownMenuRadioGroup value={locale} onValueChange={setLocale}>
      {locales.map((value) => (
        <DropdownMenuRadioItem key={value} lang={value} value={value}>
          <span aria-hidden>{localeConfig[value].flag}</span>
          {localeConfig[value].name}
        </DropdownMenuRadioItem>
      ))}
    </DropdownMenuRadioGroup>
  );
}

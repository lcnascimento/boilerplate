"use client";

import { Monitor } from "lucide-react";
import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";

import { Button } from "@boilerplate/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@boilerplate/ui/components/dropdown-menu";

import { useMounted } from "@/shared/hooks/use-mounted";
import { parseTheme, themeOptions } from "@/shared/theme";

import { ThemeRadioItems } from "./menu-radio-items";

export function ThemeSwitcher() {
  const t = useTranslations("profile");
  const { theme } = useTheme();
  const mounted = useMounted();
  const Icon = mounted ? themeOptions[parseTheme(theme)].icon : Monitor;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label={t("theme")} size="icon-sm" variant="outline">
          <Icon aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <ThemeRadioItems />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

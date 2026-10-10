"use client";

import { Languages } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@boilerplate/ui/components/dropdown-menu";

import { LocaleRadioItems } from "./menu-radio-items";

export function LocaleSwitcher() {
  const t = useTranslations("profile");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label={t("language")} size="icon-sm" variant="outline">
          <Languages aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <LocaleRadioItems />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

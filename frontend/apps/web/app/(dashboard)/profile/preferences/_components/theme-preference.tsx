"use client";

import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";

import { Label } from "@boilerplate/ui/components/label";
import {
  RadioGroup,
  RadioGroupItem,
} from "@boilerplate/ui/components/radio-group";

import { useMounted } from "@/shared/hooks/use-mounted";
import { parseTheme, themeOptions, themes } from "@/shared/theme";

interface ThemePreferenceProps {
  labelledBy: string;
}

export function ThemePreference({ labelledBy }: ThemePreferenceProps) {
  const t = useTranslations("profile");
  const { theme, setTheme } = useTheme();
  // The stored theme is only known in the browser; until then nothing is
  // checked, so the server HTML never shows a wrong choice.
  const mounted = useMounted();

  return (
    <RadioGroup
      aria-labelledby={labelledBy}
      className="grid gap-3 sm:grid-cols-3"
      value={mounted ? parseTheme(theme) : ""}
      onValueChange={setTheme}
    >
      {themes.map((value) => {
        const { labelKey, icon: Icon } = themeOptions[value];
        const id = `theme-${value}`;
        return (
          <Label
            key={value}
            className="flex cursor-pointer items-center gap-3 rounded-md border p-3 has-data-[state=checked]:border-primary"
            htmlFor={id}
          >
            <RadioGroupItem id={id} value={value} />
            <Icon aria-hidden className="size-4" />
            {t(labelKey)}
          </Label>
        );
      })}
    </RadioGroup>
  );
}

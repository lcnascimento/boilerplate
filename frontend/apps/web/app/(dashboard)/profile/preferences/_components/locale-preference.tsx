"use client";

import { localeConfig, locales } from "@boilerplate/i18n";
import { Label } from "@boilerplate/ui/components/label";
import {
  RadioGroup,
  RadioGroupItem,
} from "@boilerplate/ui/components/radio-group";

import { useLocaleSwitch } from "@/shared/hooks/use-locale-switch";

interface LocalePreferenceProps {
  labelledBy: string;
}

export function LocalePreference({ labelledBy }: LocalePreferenceProps) {
  const { locale, setLocale, isPending } = useLocaleSwitch();

  return (
    <RadioGroup
      aria-busy={isPending}
      aria-labelledby={labelledBy}
      className="grid gap-3 sm:grid-cols-3"
      disabled={isPending}
      value={locale}
      onValueChange={setLocale}
    >
      {locales.map((value) => {
        const id = `locale-${value}`;
        return (
          <Label
            key={value}
            className="flex cursor-pointer items-center gap-3 rounded-md border p-3 has-data-[state=checked]:border-primary"
            htmlFor={id}
            lang={value}
          >
            <RadioGroupItem id={id} value={value} />
            <span aria-hidden>{localeConfig[value].flag}</span>
            {localeConfig[value].name}
          </Label>
        );
      })}
    </RadioGroup>
  );
}

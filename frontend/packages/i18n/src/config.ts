export const locales = ["pt-BR", "en"] as const;

export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = "pt-BR";

export const localeConfig: Record<Locale, { name: string; flag: string }> = {
  "pt-BR": { name: "Português", flag: "🇧🇷" },
  en: { name: "English", flag: "🇺🇸" },
};

export const LOCALE_COOKIE_NAME = "boilerplate-locale";

export function isLocale(value: unknown): value is Locale {
  return locales.some((locale) => locale === value);
}

export function parseLocale(value: unknown): Locale {
  return isLocale(value) ? value : defaultLocale;
}

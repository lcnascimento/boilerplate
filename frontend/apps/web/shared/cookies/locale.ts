import { LOCALE_COOKIE_NAME, type Locale } from "@boilerplate/i18n";

export { LOCALE_COOKIE_NAME };

export const LOCALE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

export function writeLocaleCookie(locale: Locale): void {
  document.cookie = `${LOCALE_COOKIE_NAME}=${locale}; path=/; max-age=${LOCALE_COOKIE_MAX_AGE}; SameSite=Lax`;
}

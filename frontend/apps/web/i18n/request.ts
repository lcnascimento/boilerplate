import { cookies } from "next/headers";
import { getRequestConfig } from "next-intl/server";

import { messages, parseLocale } from "@boilerplate/i18n";

import { appMessages } from "@/shared/_messages";
import { LOCALE_COOKIE_NAME } from "@/shared/cookies";

// No locale routing: the locale comes from the cookie, else the default.
export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const locale = parseLocale(cookieStore.get(LOCALE_COOKIE_NAME)?.value);

  return {
    locale,
    messages: { ...messages[locale], ...appMessages(locale) },
  };
});

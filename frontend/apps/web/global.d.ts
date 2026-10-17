import type { AppMessages } from "@/shared/_messages";
import type { Locale, Messages } from "@boilerplate/i18n";

declare module "next-intl" {
  interface AppConfig {
    Locale: Locale;
    Messages: Messages & AppMessages;
  }
}

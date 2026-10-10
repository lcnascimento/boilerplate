import type { AppMessages } from "@/shared/_messages";
import type { Locale, Messages } from "@boilerplate/i18n";

// Types every `useTranslations`/`getTranslations` call: a key missing from the
// global or app catalogues fails the typecheck.
declare module "next-intl" {
  interface AppConfig {
    Locale: Locale;
    Messages: Messages & AppMessages;
  }
}

import nav from "./nav.json" with { type: "json" };
import profile from "./profile.json" with { type: "json" };

import type { Locale } from "@boilerplate/i18n";

/**
 * App namespaces, one file per namespace holding every locale. This function
 * is the list of namespaces: add a namespace by importing its file and adding
 * one line here.
 */
export function appMessages(locale: Locale) {
  return {
    nav: nav[locale],
    profile: profile[locale],
  };
}

type Namespaces = ReturnType<typeof appMessages>;

/**
 * Each namespace keeps only the keys every locale has (`Pick` over the union of
 * locales), so a key missing from any locale fails the typecheck where used.
 */
export type AppMessages = {
  [N in keyof Namespaces]: Pick<Namespaces[N], keyof Namespaces[N]>;
};

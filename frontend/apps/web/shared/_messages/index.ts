import auth from "./auth.json" with { type: "json" };
import members from "./members.json" with { type: "json" };
import nav from "./nav.json" with { type: "json" };
// example:begin
import notes from "./notes.json" with { type: "json" };
// example:end
import profile from "./profile.json" with { type: "json" };

import type { Locale } from "@boilerplate/i18n";

export function appMessages(locale: Locale) {
  return {
    auth: auth[locale],
    members: members[locale],
    nav: nav[locale],
    // example:begin
    notes: notes[locale],
    // example:end
    profile: profile[locale],
  };
}

type Namespaces = ReturnType<typeof appMessages>;

export type AppMessages = {
  [N in keyof Namespaces]: Pick<Namespaces[N], keyof Namespaces[N]>;
};

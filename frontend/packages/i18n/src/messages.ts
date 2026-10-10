import en from "./messages/en.json" with { type: "json" };
import ptBR from "./messages/pt-BR.json" with { type: "json" };

import type { Locale } from "./config";

/** Shape of the global catalogue; pt-BR (the default locale) is the source. */
export type Messages = typeof ptBR;

/** Global namespaces (`common`, `errors`, `validation`) per locale. */
export const messages = {
  "pt-BR": ptBR,
  en,
} satisfies Record<Locale, Messages>;

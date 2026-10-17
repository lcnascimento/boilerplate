import en from "./messages/en.json" with { type: "json" };
import ptBR from "./messages/pt-BR.json" with { type: "json" };

import type { Locale } from "./config";

export type Messages = typeof ptBR;

export const messages = {
  "pt-BR": ptBR,
  en,
} satisfies Record<Locale, Messages>;

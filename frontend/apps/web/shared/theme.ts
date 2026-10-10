import { Monitor, Moon, Sun, type LucideIcon } from "lucide-react";

import { themes, type Theme } from "@boilerplate/types";

import type { AppMessages } from "@/shared/_messages";

export { themes };
export type { Theme };

export function isTheme(value: unknown): value is Theme {
  return themes.some((theme) => theme === value);
}

/** next-themes reports `undefined` before it mounts; that reads as "system". */
export function parseTheme(value: unknown): Theme {
  return isTheme(value) ? value : "system";
}

export const themeOptions = {
  light: { labelKey: "themeLight", icon: Sun },
  dark: { labelKey: "themeDark", icon: Moon },
  system: { labelKey: "themeSystem", icon: Monitor },
} as const satisfies Record<
  Theme,
  { labelKey: keyof AppMessages["profile"]; icon: LucideIcon }
>;

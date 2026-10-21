import {
  House,
  // example:begin
  NotebookPen,
  // example:end
  SlidersHorizontal,
  Users,
  type LucideIcon,
} from "lucide-react";

import type { AppMessages } from "@/shared/_messages";
import type { Route } from "next";

// example:begin
export const NOTES_HREF = "/notes" satisfies Route;
// example:end

type NavLabelKey = {
  [N in keyof AppMessages]: {
    [K in keyof AppMessages[N]]: AppMessages[N][K] extends string
      ? `${N}.${Extract<K, string>}`
      : never;
  }[keyof AppMessages[N]];
}[keyof AppMessages];

interface NavRoute {
  href: Route;
  labelKey: NavLabelKey;
  icon: LucideIcon;
  area: "sidebar" | "userMenu";
}

export const navRoutes = [
  { href: "/", labelKey: "nav.home", icon: House, area: "sidebar" },
  // example:begin
  {
    href: NOTES_HREF,
    labelKey: "notes.title",
    icon: NotebookPen,
    area: "sidebar",
  },
  // example:end
  {
    href: "/settings/members",
    labelKey: "members.title",
    icon: Users,
    area: "sidebar",
  },
  {
    href: "/profile/preferences",
    labelKey: "nav.preferences",
    icon: SlidersHorizontal,
    area: "userMenu",
  },
] as const satisfies readonly NavRoute[];

export function routesIn(area: NavRoute["area"]): NavRoute[] {
  return navRoutes.filter((route) => route.area === area);
}

export function isActiveHref(pathname: string, href: string): boolean {
  const path = trimTrailingSlash(pathname);
  if (href === "/") return path === "/";
  return path === href || path.startsWith(`${href}/`);
}

export interface Crumb {
  href: Route;
  labelKey: NavLabelKey;
  current: boolean;
}

export function breadcrumbTrail(pathname: string): Crumb[] {
  const path = trimTrailingSlash(pathname);
  const home: Crumb = {
    href: "/",
    labelKey: "nav.home",
    current: path === "/",
  };
  const matched = navRoutes
    .filter((route) => route.href !== "/" && isActiveHref(path, route.href))
    .sort((a, b) => a.href.length - b.href.length)
    .map((route): Crumb => ({
      href: route.href,
      labelKey: route.labelKey,
      current: route.href === path,
    }));
  return [home, ...matched];
}

function trimTrailingSlash(pathname: string): string {
  return pathname.length > 1 && pathname.endsWith("/")
    ? pathname.slice(0, -1)
    : pathname;
}

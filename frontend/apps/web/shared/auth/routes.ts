// example:begin
import { NOTES_HREF } from "@/shared/navigation";
// example:end

import type { Route } from "next";

export const SIGN_IN_HREF = "/sign-in" satisfies Route;

export const SIGN_UP_HREF = "/sign-up" satisfies Route;

export const CALLBACK_URL_PARAM = "callbackUrl";

const signedInHomes: readonly Route[] = [
  // example:begin
  NOTES_HREF,
  // example:end
];

export const SIGNED_IN_HOME: Route = signedInHomes[0] ?? "/";

const authPaths: readonly string[] = [SIGN_IN_HREF, SIGN_UP_HREF];

export function isAuthPath(pathname: string): boolean {
  return authPaths.includes(pathname);
}

export function signInHrefFor(callbackUrl: string): string {
  const params = new URLSearchParams({ [CALLBACK_URL_PARAM]: callbackUrl });
  return `${SIGN_IN_HREF}?${params.toString()}`;
}

export function parseCallbackUrl(raw: string | null | undefined): Route {
  if (!raw?.startsWith("/") || raw.startsWith("//") || raw.includes("\\")) {
    return SIGNED_IN_HOME;
  }
  // Next typedRoutes types a non-literal path only through `as Route`.
  return raw as Route;
}

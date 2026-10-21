import { getSessionCookie } from "better-auth/cookies";
import { NextResponse, type NextRequest } from "next/server";

import { getIam } from "@/shared/auth/auth";
import {
  isAuthPath,
  SIGNED_IN_HOME,
  signInHrefFor,
} from "@/shared/auth/routes";

async function hasValidSession(request: NextRequest): Promise<boolean> {
  const session = await getIam().auth.api.getSession({
    headers: request.headers,
  });
  return session !== null;
}

function hasSessionCookie(request: NextRequest): boolean {
  return getSessionCookie(request) !== null;
}

export async function proxy(request: NextRequest): Promise<NextResponse> {
  const { pathname, search } = request.nextUrl;

  if (isAuthPath(pathname)) {
    return (await hasValidSession(request))
      ? NextResponse.redirect(new URL(SIGNED_IN_HOME, request.url))
      : NextResponse.next();
  }

  return hasSessionCookie(request)
    ? NextResponse.next()
    : NextResponse.redirect(
        new URL(signInHrefFor(`${pathname}${search}`), request.url)
      );
}

export const config = {
  matcher: [
    "/((?!api/|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)",
  ],
};

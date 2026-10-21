import { getSessionCookie } from "better-auth/cookies";
import { NextResponse, type NextRequest } from "next/server";

import { getIam } from "@/shared/auth/auth";
import {
  isAuthPath,
  SIGNED_IN_HOME,
  signInHrefFor,
} from "@/shared/auth/routes";

export async function proxy(request: NextRequest): Promise<NextResponse> {
  const { pathname, search } = request.nextUrl;

  if (isAuthPath(pathname)) {
    // A stale cookie must not bounce between here and the (dashboard) layout, so this checks the session itself.
    const session = await getIam().auth.api.getSession({
      headers: request.headers,
    });
    return session === null
      ? NextResponse.next()
      : NextResponse.redirect(new URL(SIGNED_IN_HOME, request.url));
  }

  // Only the cookie's presence: the (dashboard) layout validates the session.
  return getSessionCookie(request) === null
    ? NextResponse.redirect(
        new URL(signInHrefFor(`${pathname}${search}`), request.url)
      )
    : NextResponse.next();
}

export const config = {
  matcher: [
    "/((?!api/|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)",
  ],
};

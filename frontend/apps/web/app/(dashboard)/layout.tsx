import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { getIam } from "@/shared/auth/auth";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { getRequestSession } from "@/shared/auth/session";
import { parseSidebarOpen, SIDEBAR_COOKIE_NAME } from "@/shared/cookies";

import { DashboardShell, NoOrganization } from "./_components";

import type { ReactNode } from "react";

interface DashboardLayoutProps {
  children: ReactNode;
}

// Next answers notFound() with a 404 only before streaming starts, so this group has no Suspense or loading.tsx.
export default async function DashboardLayout({
  children,
}: DashboardLayoutProps) {
  const [cookieStore, session] = await Promise.all([
    cookies(),
    getRequestSession(),
  ]);
  if (session === null) redirect(SIGN_IN_HREF);

  const { id, name, email, image } = session.user;
  await getIam().ensureSignUpOrganization(id);
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind === "no-organization") return <NoOrganization />;

  return (
    <DashboardShell
      defaultOpen={parseSidebarOpen(
        cookieStore.get(SIDEBAR_COOKIE_NAME)?.value
      )}
      organizations={requestActor.organizations}
      user={{ id, name, email, avatarUrl: image ?? undefined }}
    >
      {children}
    </DashboardShell>
  );
}

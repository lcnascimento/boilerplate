import { cookies } from "next/headers";

import { parseSidebarOpen, SIDEBAR_COOKIE_NAME } from "@/shared/cookies";
import { getLocalUser } from "@/shared/local-user";

import { DashboardShell } from "./_components";

import type { ReactNode } from "react";

interface DashboardLayoutProps {
  children: ReactNode;
}

// Next answers notFound() with a 404 only before streaming starts, so this group has no Suspense or loading.tsx.
export default async function DashboardLayout({
  children,
}: DashboardLayoutProps) {
  const [cookieStore, user] = await Promise.all([cookies(), getLocalUser()]);

  return (
    <DashboardShell
      defaultOpen={parseSidebarOpen(
        cookieStore.get(SIDEBAR_COOKIE_NAME)?.value
      )}
      user={user}
    >
      {children}
    </DashboardShell>
  );
}

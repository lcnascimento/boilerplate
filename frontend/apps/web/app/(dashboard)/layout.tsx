import { cookies } from "next/headers";

import { parseSidebarOpen, SIDEBAR_COOKIE_NAME } from "@/shared/cookies";
import { getLocalUser } from "@/shared/local-user";

import { DashboardShell } from "./_components";

import type { ReactNode } from "react";

interface DashboardLayoutProps {
  children: ReactNode;
}

// No Suspense boundary and no loading.tsx in this group: a page that calls
// notFound() before streaming starts answers a real 404. Pages stream their
// own slow parts behind a local Suspense.
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

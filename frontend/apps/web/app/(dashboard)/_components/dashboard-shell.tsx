import { cookies } from "next/headers";

import {
  SidebarInset,
  SidebarProvider,
} from "@boilerplate/ui/components/sidebar";

import { NavHeader, NavSidebar } from "@/shared/components";
import { parseSidebarOpen, SIDEBAR_COOKIE_NAME } from "@/shared/cookies";
import { LOCAL_USER } from "@/shared/local-user";

import type { ReactNode } from "react";

interface DashboardShellProps {
  children: ReactNode;
}

/**
 * The app frame: sidebar, header and content. It fetches no data; it only
 * reads the `sidebar_state` cookie, so the sidebar renders collapsed or open
 * on the server and doesn't flash after a reload.
 */
export async function DashboardShell({ children }: DashboardShellProps) {
  const cookieStore = await cookies();
  const defaultOpen = parseSidebarOpen(
    cookieStore.get(SIDEBAR_COOKIE_NAME)?.value
  );

  return (
    <SidebarProvider defaultOpen={defaultOpen}>
      <NavSidebar user={LOCAL_USER} />
      <SidebarInset>
        <NavHeader />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  );
}

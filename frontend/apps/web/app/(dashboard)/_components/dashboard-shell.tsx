import {
  SidebarInset,
  SidebarProvider,
} from "@boilerplate/ui/components/sidebar";

import { NavHeader, NavSidebar } from "@/shared/components";

import type { User } from "@boilerplate/types";
import type { ReactNode } from "react";

interface DashboardShellProps {
  /** Whether the sidebar starts open, from the `sidebar_state` cookie. */
  defaultOpen: boolean;
  user: User;
  children: ReactNode;
}

/**
 * The app frame: sidebar, header and content. It is synchronous and fetches
 * nothing, so nothing streams before the page renders and a page's
 * `notFound()` still answers 404.
 */
export function DashboardShell({
  defaultOpen,
  user,
  children,
}: DashboardShellProps) {
  return (
    <SidebarProvider defaultOpen={defaultOpen}>
      <NavSidebar user={user} />
      <SidebarInset>
        <NavHeader />
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  );
}

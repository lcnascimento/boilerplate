import {
  SidebarInset,
  SidebarProvider,
} from "@boilerplate/ui/components/sidebar";

import { OrganizationsProvider } from "@/providers";
import { NavHeader, NavSidebar } from "@/shared/components";

import type { UserOrganizations } from "@/shared/auth/organizations";
import type { User } from "@boilerplate/types";
import type { ReactNode } from "react";

interface DashboardShellProps {
  defaultOpen: boolean;
  user: User;
  organizations: UserOrganizations;
  children: ReactNode;
}

export function DashboardShell({
  defaultOpen,
  user,
  organizations,
  children,
}: DashboardShellProps) {
  return (
    <OrganizationsProvider organizations={organizations}>
      <SidebarProvider defaultOpen={defaultOpen}>
        <NavSidebar user={user} />
        <SidebarInset>
          <NavHeader />
          <div className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</div>
        </SidebarInset>
      </SidebarProvider>
    </OrganizationsProvider>
  );
}

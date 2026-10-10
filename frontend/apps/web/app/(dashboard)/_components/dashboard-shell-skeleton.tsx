import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarMenuSkeleton,
  SidebarProvider,
} from "@boilerplate/ui/components/sidebar";
import { Skeleton } from "@boilerplate/ui/components/skeleton";

import { ContentSkeleton } from "./content-skeleton";

/** Mirrors `DashboardShell` while it reads the sidebar cookie. */
export function DashboardShellSkeleton() {
  return (
    <SidebarProvider>
      <Sidebar collapsible="icon" variant="inset">
        <SidebarHeader>
          <div className="flex items-center gap-2 p-2">
            <Skeleton className="size-8 rounded-lg" />
            <Skeleton className="h-4 w-24" />
          </div>
        </SidebarHeader>
        <SidebarContent className="p-2">
          <SidebarMenuSkeleton showIcon />
        </SidebarContent>
        <SidebarFooter>
          <div className="flex items-center gap-2 p-2">
            <Skeleton className="size-8 rounded-lg" />
            <Skeleton className="h-4 w-28" />
          </div>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 px-4">
          <Skeleton className="size-7" />
          <Skeleton className="h-4 w-32" />
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <ContentSkeleton />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

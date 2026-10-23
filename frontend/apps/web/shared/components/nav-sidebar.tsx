"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@boilerplate/ui/components/sidebar";

import { isActiveHref, routesIn } from "@/shared/navigation";

import { NavUser } from "./nav-user";
import { OrganizationSwitcher } from "./organization-switcher";

import type { User } from "@boilerplate/types";

interface NavSidebarProps {
  user: User;
}

export function NavSidebar({ user }: NavSidebarProps) {
  const t = useTranslations();
  const pathname = usePathname();

  return (
    <Sidebar collapsible="icon" variant="inset">
      <SidebarHeader>
        <OrganizationSwitcher />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{t("nav.navigation")}</SidebarGroupLabel>
          <SidebarMenu>
            {routesIn("sidebar").map(({ href, labelKey, icon: Icon }) => (
              <SidebarMenuItem key={href}>
                <SidebarMenuButton
                  asChild
                  isActive={isActiveHref(pathname, href)}
                  tooltip={t(labelKey)}
                >
                  <Link href={href}>
                    <Icon aria-hidden />
                    <span>{t(labelKey)}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

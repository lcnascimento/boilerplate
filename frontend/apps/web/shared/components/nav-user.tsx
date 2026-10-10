"use client";

import { ChevronsUpDown, Languages, Palette } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";

import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@boilerplate/ui/components/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@boilerplate/ui/components/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@boilerplate/ui/components/sidebar";
import { getInitials } from "@boilerplate/utils";

import { routesIn } from "@/shared/navigation";

import { LocaleRadioItems, ThemeRadioItems } from "./menu-radio-items";

import type { User } from "@boilerplate/types";

interface NavUserProps {
  user: User;
}

function UserIdentity({ user }: NavUserProps) {
  return (
    <>
      <Avatar className="size-8 rounded-lg">
        {user.avatarUrl ? (
          <AvatarImage alt={user.name} src={user.avatarUrl} />
        ) : null}
        <AvatarFallback className="rounded-lg">
          {getInitials(user.name)}
        </AvatarFallback>
      </Avatar>
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-medium">{user.name}</span>
        {user.email ? (
          <span className="truncate text-xs">{user.email}</span>
        ) : null}
      </div>
    </>
  );
}

export function NavUser({ user }: NavUserProps) {
  const tNav = useTranslations("nav");
  const tProfile = useTranslations("profile");
  const { isMobile } = useSidebar();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              aria-label={tNav("openUserMenu")}
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              size="lg"
            >
              <UserIdentity user={user} />
              <ChevronsUpDown aria-hidden className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>

          <DropdownMenuContent
            align="end"
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuLabel className="flex items-center gap-2 p-1 font-normal">
              <UserIdentity user={user} />
            </DropdownMenuLabel>
            <DropdownMenuSeparator />

            <DropdownMenuGroup>
              {routesIn("userMenu").map(({ href, labelKey, icon: Icon }) => (
                <DropdownMenuItem key={href} asChild>
                  <Link href={href}>
                    <Icon aria-hidden />
                    {tNav(labelKey)}
                  </Link>
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
            <DropdownMenuSeparator />

            <DropdownMenuGroup>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Palette aria-hidden />
                  {tProfile("theme")}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent>
                  <ThemeRadioItems />
                </DropdownMenuSubContent>
              </DropdownMenuSub>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Languages aria-hidden />
                  {tProfile("language")}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent>
                  <LocaleRadioItems />
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

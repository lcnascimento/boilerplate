"use client";

import { Boxes, ChevronsUpDown } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@boilerplate/ui/components/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@boilerplate/ui/components/sidebar";

import { authClient } from "@/shared/auth/client";

import type { UserOrganizations } from "@/shared/auth/organizations";

const APP_NAME = "Boilerplate";

interface OrganizationSwitcherProps {
  organizations: UserOrganizations;
}

export function OrganizationSwitcher({
  organizations: { active, organizations },
}: OrganizationSwitcherProps) {
  const t = useTranslations("organizations");
  const { isMobile } = useSidebar();
  const router = useRouter();
  const [isSwitching, startSwitch] = useTransition();

  function handleSwitch(organizationId: string) {
    if (organizationId === active.id) return;

    startSwitch(async () => {
      const { error } = await authClient.organization.setActive({
        organizationId,
      });
      if (error) {
        toast.error(t("switchFailed"));
        return;
      }
      router.refresh();
    });
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              aria-busy={isSwitching}
              aria-label={t("switcherLabel")}
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              size="lg"
            >
              <span className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <Boxes aria-hidden className="size-4" />
              </span>
              <span className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{active.name}</span>
                <span className="truncate text-xs">{APP_NAME}</span>
              </span>
              <ChevronsUpDown aria-hidden className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>

          <DropdownMenuContent
            align="start"
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              {t("listLabel")}
            </DropdownMenuLabel>
            <DropdownMenuRadioGroup
              value={active.id}
              onValueChange={handleSwitch}
            >
              {organizations.map(({ id, name }) => (
                <DropdownMenuRadioItem
                  key={id}
                  disabled={isSwitching}
                  value={id}
                >
                  {name}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

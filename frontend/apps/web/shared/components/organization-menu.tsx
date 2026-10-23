"use client";

import { ArrowLeftRight, Boxes, ChevronsUpDown, Plus } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
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

import { useOrganizations } from "@/providers";
import { authClient } from "@/shared/auth/client";
import { CREATE_OR_JOIN_HREF } from "@/shared/auth/routes";
import { routesIn } from "@/shared/navigation";

const APP_NAME = "Boilerplate";

export function OrganizationMenu() {
  const { active, organizations } = useOrganizations();
  const tAll = useTranslations();
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
              aria-label={t("menuLabel")}
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
            <DropdownMenuGroup>
              {routesIn("organization").map(
                ({ href, labelKey, icon: Icon }) => (
                  <DropdownMenuItem key={href} asChild>
                    <Link href={href}>
                      <Icon aria-hidden />
                      {tAll(labelKey)}
                    </Link>
                  </DropdownMenuItem>
                )
              )}
            </DropdownMenuGroup>
            <DropdownMenuSeparator />

            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <ArrowLeftRight aria-hidden />
                {t("switch")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="min-w-48">
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
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <Link href={CREATE_OR_JOIN_HREF}>
                    <Plus aria-hidden />
                    {t("createOrJoin")}
                  </Link>
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

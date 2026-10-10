import { useTranslations } from "next-intl";

import { Separator } from "@boilerplate/ui/components/separator";
import { SidebarTrigger } from "@boilerplate/ui/components/sidebar";

import { LocaleSwitcher } from "./locale-switcher";
import { NavBreadcrumb } from "./nav-breadcrumb";
import { ThemeSwitcher } from "./theme-switcher";

export function NavHeader() {
  const t = useTranslations("nav");

  return (
    <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
      <div className="flex w-full items-center gap-2 px-4">
        <SidebarTrigger aria-label={t("toggleSidebar")} className="-ml-1" />
        <Separator
          className="mr-2 data-[orientation=vertical]:h-4"
          orientation="vertical"
        />
        <NavBreadcrumb />
        <div className="ml-auto flex items-center gap-2">
          <LocaleSwitcher />
          <ThemeSwitcher />
        </div>
      </div>
    </header>
  );
}

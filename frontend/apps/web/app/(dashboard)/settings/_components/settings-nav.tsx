"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";

import { isActiveHref, routesIn, settingsSections } from "@/shared/navigation";

export function SettingsNav() {
  const t = useTranslations();
  const pathname = usePathname();

  return (
    <nav aria-label={t("nav.settings")} className="grid content-start gap-6">
      {settingsSections.map(({ area, labelKey }) => (
        <div key={area} className="grid gap-1">
          <h2
            className="px-2 text-xs font-medium text-muted-foreground"
            id={`settings-${area}`}
          >
            {t(labelKey)}
          </h2>
          <ul aria-labelledby={`settings-${area}`} className="grid gap-1">
            {routesIn(area).map(
              ({ href, labelKey: routeLabelKey, icon: Icon }) => (
                <li key={href}>
                  <Link
                    aria-current={
                      isActiveHref(pathname, href) ? "page" : undefined
                    }
                    className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground aria-[current=page]:bg-accent aria-[current=page]:font-medium aria-[current=page]:text-accent-foreground [&_svg]:size-4"
                    href={href}
                  >
                    <Icon aria-hidden />
                    {t(routeLabelKey)}
                  </Link>
                </li>
              )
            )}
          </ul>
        </div>
      ))}
    </nav>
  );
}

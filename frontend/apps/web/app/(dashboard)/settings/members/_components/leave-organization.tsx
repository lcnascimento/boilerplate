import { LogOut } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";

import { LeaveOrganizationDialog } from "./leave-organization-dialog";

import type { MemberPolicy } from "@/shared/auth/members";

interface LeaveOrganizationProps {
  leave: MemberPolicy["leave"];
}

const OWNER_HINT_ID = "leave-organization-hint";

export function LeaveOrganization({ leave }: LeaveOrganizationProps) {
  const t = useTranslations("members");

  if (leave === "allowed") return <LeaveOrganizationDialog />;

  return (
    <div className="flex flex-col items-start gap-2">
      <Button aria-describedby={OWNER_HINT_ID} disabled variant="outline">
        <LogOut aria-hidden />
        {t("leave")}
      </Button>
      <p className="text-sm text-muted-foreground" id={OWNER_HINT_ID}>
        {t("ownerMustTransferBeforeLeaving")}
      </p>
      <Link
        className="text-sm font-medium underline underline-offset-4"
        href="/settings/organization"
      >
        {t("transferOwnership")}
      </Link>
    </div>
  );
}

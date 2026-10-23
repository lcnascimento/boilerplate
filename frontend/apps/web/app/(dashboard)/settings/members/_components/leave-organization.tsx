import { LogOut } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";

import { LeaveOrganizationDialog } from "./leave-organization-dialog";

import type { MemberPolicy } from "@/shared/auth/members";

interface LeaveOrganizationProps {
  leave: MemberPolicy["leave"];
}

const LAST_OWNER_HINT_ID = "leave-organization-hint";

export function LeaveOrganization({ leave }: LeaveOrganizationProps) {
  const t = useTranslations("members");

  if (leave === "allowed") return <LeaveOrganizationDialog />;

  return (
    <div className="flex flex-col items-start gap-2">
      <Button aria-describedby={LAST_OWNER_HINT_ID} disabled variant="outline">
        <LogOut aria-hidden />
        {t("leave")}
      </Button>
      <p className="text-sm text-muted-foreground" id={LAST_OWNER_HINT_ID}>
        {t("lastOwnerCannotLeave")}
      </p>
    </div>
  );
}

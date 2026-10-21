"use client";

import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type MouseEvent } from "react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@boilerplate/ui/components/alert-dialog";
import { Button } from "@boilerplate/ui/components/button";

import { leaveOrganization } from "@/actions/members";

import type { MemberPolicy } from "@/shared/auth/members";

interface LeaveOrganizationProps {
  leave: MemberPolicy["leave"];
}

const LAST_OWNER_HINT_ID = "leave-organization-hint";

export function LeaveOrganization({ leave }: LeaveOrganizationProps) {
  const t = useTranslations("members");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();

  if (leave === "lastOwner") {
    return (
      <div className="flex flex-col items-start gap-2">
        <Button
          aria-describedby={LAST_OWNER_HINT_ID}
          disabled
          variant="outline"
        >
          <LogOut aria-hidden />
          {t("leave")}
        </Button>
        <p className="text-sm text-muted-foreground" id={LAST_OWNER_HINT_ID}>
          {t("lastOwnerCannotLeave")}
        </p>
      </div>
    );
  }

  function handleConfirm(event: MouseEvent<HTMLButtonElement>) {
    // Radix closes the dialog on the action's click.
    event.preventDefault();

    startTransition(async () => {
      const result = await leaveOrganization();
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`));
        return;
      }
      setOpen(false);
      router.push("/");
      router.refresh();
    });
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        <Button variant="outline">
          <LogOut aria-hidden />
          {t("leave")}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("leaveTitle")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("leaveDescription")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>
            {tCommon("cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            aria-busy={isPending}
            disabled={isPending}
            variant="destructive"
            onClick={handleConfirm}
          >
            {t("leaveConfirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

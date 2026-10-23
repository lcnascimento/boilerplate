"use client";

import { ArrowLeftRight, Trash2 } from "lucide-react";
import Link from "next/link";
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
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { deleteAccount } from "@/actions/account";
import { authClient } from "@/shared/auth/client";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { useConfirmDialog } from "@/shared/hooks/use-confirm-dialog";

import type { Organization } from "@/shared/auth/organizations";
import type { Route } from "next";

const ORGANIZATION_SETTINGS_HREF = "/settings/organization" satisfies Route;
const PASSWORD_ID = "delete-account-password";

interface DeleteAccountProps {
  blockers: readonly Organization[];
}

export function DeleteAccount({ blockers }: DeleteAccountProps) {
  return blockers.length > 0 ? (
    <DeletionBlockers blockers={blockers} />
  ) : (
    <DeleteAccountDialog />
  );
}

function DeletionBlockers({ blockers }: DeleteAccountProps) {
  const t = useTranslations("accountSettings");
  const router = useRouter();
  const [isSwitching, startSwitch] = useTransition();

  function openTransfer(
    event: MouseEvent<HTMLAnchorElement>,
    organization: Organization
  ) {
    event.preventDefault();
    startSwitch(async () => {
      const { error } = await authClient.organization.setActive({
        organizationId: organization.id,
      });
      if (error) {
        toast.error(t("switchFailed", { name: organization.name }));
        return;
      }
      router.push(ORGANIZATION_SETTINGS_HREF);
      router.refresh();
    });
  }

  return (
    <div className="grid gap-3">
      <p className="text-sm">{t("blockedDescription")}</p>
      <ul
        aria-busy={isSwitching}
        aria-label={t("blockersLabel")}
        className="grid gap-2"
      >
        {blockers.map((organization) => (
          <li key={organization.id} className="flex items-center gap-2 text-sm">
            <span className="font-medium">{organization.name}</span>
            <Link
              className="inline-flex items-center gap-1 text-primary underline-offset-4 hover:underline [&_svg]:size-4"
              href={ORGANIZATION_SETTINGS_HREF}
              onClick={(event) => openTransfer(event, organization)}
            >
              <ArrowLeftRight aria-hidden />
              {t("transferOwnership", { name: organization.name })}
            </Link>
          </li>
        ))}
      </ul>
      <Button disabled className="justify-self-start" variant="destructive">
        <Trash2 aria-hidden />
        {t("delete")}
      </Button>
    </div>
  );
}

function DeleteAccountDialog() {
  const t = useTranslations("accountSettings");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [password, setPassword] = useState("");

  const { open, setOpen, isPending, handleConfirm } = useConfirmDialog(
    async () => {
      const result = await deleteAccount({ password });
      if (result.kind === "error") {
        toast.error(t(`deleteErrors.${result.code}`));
        if (result.code === "ownsSharedOrganizations") router.refresh();
        return "failed";
      }
      toast.success(t("deleted"));
      router.replace(SIGN_IN_HREF);
      router.refresh();
      return "done";
    }
  );

  function handleOpenChange(next: boolean) {
    if (!next) setPassword("");
    setOpen(next);
  }

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogTrigger asChild>
        <Button className="justify-self-start" variant="destructive">
          <Trash2 aria-hidden />
          {t("delete")}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("deleteConfirmTitle")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("deleteConfirmDescription")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="grid gap-2">
          <Label htmlFor={PASSWORD_ID}>{t("deleteConfirmLabel")}</Label>
          <Input
            autoComplete="current-password"
            id={PASSWORD_ID}
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>
            {tCommon("cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            aria-busy={isPending}
            disabled={isPending || password === ""}
            variant="destructive"
            onClick={handleConfirm}
          >
            {t("deleteConfirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

"use client";

import { Crown } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState } from "react";
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
import { Label } from "@boilerplate/ui/components/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@boilerplate/ui/components/select";

import { transferOwnership } from "@/actions/organization";
import { useConfirmDialog } from "@/shared/hooks/use-confirm-dialog";

import type { Member, OwnershipPolicy } from "@/shared/auth/members";

interface TransferOwnershipProps {
  policy: OwnershipPolicy;
}

export function TransferOwnership({ policy }: TransferOwnershipProps) {
  const t = useTranslations("organizationSettings");

  if (policy.kind === "ownedBy") {
    return (
      <p className="text-sm text-muted-foreground">
        {t("ownedBy", { name: policy.owner.name })}
      </p>
    );
  }

  if (policy.candidates.length === 0) {
    return <p className="text-sm text-muted-foreground">{t("noCandidates")}</p>;
  }

  return <TransferOwnershipForm candidates={policy.candidates} />;
}

interface TransferOwnershipFormProps {
  candidates: Member[];
}

function TransferOwnershipForm({ candidates }: TransferOwnershipFormProps) {
  const t = useTranslations("organizationSettings");
  const tRoles = useTranslations("members.roles");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [memberId, setMemberId] = useState<string>();
  const chosen = candidates.find(({ id }) => id === memberId);

  const { open, setOpen, isPending, handleConfirm } = useConfirmDialog(
    async () => {
      if (chosen === undefined) return "failed";
      const result = await transferOwnership({ memberId: chosen.id });
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`));
        return "failed";
      }
      toast.success(t("transferred", { name: chosen.name }));
      router.refresh();
      return "done";
    }
  );

  return (
    <div className="grid max-w-sm gap-4">
      <div className="grid gap-2">
        <Label htmlFor="new-owner">{t("newOwnerLabel")}</Label>
        <Select value={memberId} onValueChange={setMemberId}>
          <SelectTrigger className="w-full" id="new-owner">
            <SelectValue placeholder={t("newOwnerPlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {candidates.map(({ id, name, role }) => (
              <SelectItem key={id} value={id}>
                {t("candidate", { name, role: tRoles(role) })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogTrigger asChild>
          <Button
            className="justify-self-start"
            disabled={chosen === undefined}
            variant="destructive"
          >
            <Crown aria-hidden />
            {t("transfer")}
          </Button>
        </AlertDialogTrigger>
        {chosen === undefined ? null : (
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t("transferConfirmTitle", { name: chosen.name })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t("transferConfirmDescription", { name: chosen.name })}
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
                {t("transfer")}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        )}
      </AlertDialog>
    </div>
  );
}

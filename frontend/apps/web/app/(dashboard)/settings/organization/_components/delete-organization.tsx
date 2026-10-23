"use client";

import { Trash2 } from "lucide-react";
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
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { deleteOrganization } from "@/actions/organization";
import { useConfirmDialog } from "@/shared/hooks/use-confirm-dialog";

interface DeleteOrganizationProps {
  name: string;
}

const CONFIRMATION_ID = "delete-organization-confirmation";

export function DeleteOrganization({ name }: DeleteOrganizationProps) {
  const t = useTranslations("organizationSettings");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [confirmation, setConfirmation] = useState("");

  const { open, setOpen, isPending, handleConfirm } = useConfirmDialog(
    async () => {
      const result = await deleteOrganization({ confirmation });
      if (result.kind === "error") {
        toast.error(t(`deleteErrors.${result.code}`));
        return "failed";
      }
      toast.success(t("deleted", { name }));
      router.push("/");
      router.refresh();
      return "done";
    }
  );

  function handleOpenChange(next: boolean) {
    if (!next) setConfirmation("");
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
          <AlertDialogTitle>
            {t("deleteConfirmTitle", { name })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("deleteConfirmDescription")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="grid gap-2">
          <Label htmlFor={CONFIRMATION_ID}>
            {t("deleteConfirmLabel", { name })}
          </Label>
          <Input
            autoComplete="off"
            id={CONFIRMATION_ID}
            type="text"
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>
            {tCommon("cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            aria-busy={isPending}
            disabled={isPending || confirmation !== name}
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

"use client";

import { UserMinus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
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

import { removeMember } from "@/actions/members";
import { useConfirmDialog } from "@/shared/hooks/use-confirm-dialog";

import type { Member } from "@/shared/auth/members";

interface RemoveMemberDialogProps {
  member: Pick<Member, "id" | "name">;
}

export function RemoveMemberDialog({ member }: RemoveMemberDialogProps) {
  const t = useTranslations("members");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const { open, setOpen, isPending, handleConfirm } = useConfirmDialog(
    async () => {
      const result = await removeMember({ memberId: member.id });
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`));
        return "failed";
      }
      toast.success(t("removed"));
      router.refresh();
      return "done";
    }
  );

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        <Button
          aria-label={t("removeMemberOf", { name: member.name })}
          size="icon"
          variant="ghost"
        >
          <UserMinus aria-hidden />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("removeTitle", { name: member.name })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("removeDescription", { name: member.name })}
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
            {t("remove")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

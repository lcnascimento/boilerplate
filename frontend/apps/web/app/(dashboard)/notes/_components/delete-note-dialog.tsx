"use client";

import { Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type MouseEvent } from "react";
import { toast } from "sonner";

import { ErrorCode } from "@boilerplate/api/core/errors";
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

import { deleteNote } from "@/actions/notes";
import { NOTES_HREF } from "@/shared/navigation";

import type { APIError } from "@boilerplate/api/core";
import type { Note } from "@boilerplate/api/notes/schemas";

interface DeleteNoteDialogProps {
  id: Note["id"];
  title: string;
}

export function DeleteNoteDialog({ id, title }: DeleteNoteDialogProps) {
  const t = useTranslations("notes");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();

  function failureMessage(error: Pick<APIError, "code" | "status">) {
    if (error.status === 403) return t("deleteForbidden");
    if (error.code === ErrorCode.Unavailable)
      return t("unavailableDescription");
    return t("deleteFailed");
  }

  function handleConfirm(event: MouseEvent<HTMLButtonElement>) {
    // Radix closes the dialog on the action's click.
    event.preventDefault();

    startTransition(async () => {
      const result = await deleteNote(id);
      if (result.kind === "error") {
        toast.error(failureMessage(result.error));
        return;
      }
      setOpen(false);
      toast.success(t("deleted"));
      router.push(NOTES_HREF);
      router.refresh();
    });
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        <Button variant="destructive">
          <Trash2 aria-hidden />
          {tCommon("delete")}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("deleteTitle")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("deleteDescription", { title })}
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
            {tCommon("delete")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

"use client";

import { Trash2 } from "lucide-react";
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

import { deleteNote } from "@/actions/notes";

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

  function handleConfirm(event: MouseEvent<HTMLButtonElement>) {
    // Keep the dialog open until the service answers.
    event.preventDefault();

    startTransition(async () => {
      const result = await deleteNote(id);
      if (result.kind === "error") {
        toast.error(
          result.error.code === "UNAVAILABLE"
            ? t("unavailableDescription")
            : t("deleteFailed")
        );
        return;
      }
      setOpen(false);
      toast.success(t("deleted"));
      router.push("/notes");
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

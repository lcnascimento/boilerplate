"use client";

import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { toast } from "sonner";

import { ErrorCode } from "@boilerplate/api/core/errors";
import {
  createNoteInputSchema,
  type CreateNoteInput,
} from "@boilerplate/api/notes/schemas";
import { Button } from "@boilerplate/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@boilerplate/ui/components/dialog";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";
import { Textarea } from "@boilerplate/ui/components/textarea";

import { createNote } from "@/actions/notes";

import type { APIError } from "@boilerplate/api/core";

type FieldErrors = Partial<Record<keyof CreateNoteInput, string>>;

const fieldSchema = createNoteInputSchema.keyof();

export function CreateNoteDialog() {
  const t = useTranslations("notes");
  const tCommon = useTranslations("common");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function reset() {
    setTitle("");
    setBody("");
    setFieldErrors({});
    setFormError(undefined);
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset();
    setOpen(next);
  }

  function failureMessage(error: Pick<APIError, "code" | "status">) {
    if (error.status === 403) return t("createForbidden");
    if (error.code === ErrorCode.Unavailable)
      return t("unavailableDescription");
    return t("createFailed");
  }

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(undefined);

    const parsed = createNoteInputSchema.safeParse({
      title,
      body: body === "" ? undefined : body,
    });
    if (!parsed.success) {
      const errors: FieldErrors = {};
      for (const issue of parsed.error.issues) {
        const field = fieldSchema.safeParse(issue.path[0]);
        if (!field.success || errors[field.data] !== undefined) continue;
        errors[field.data] =
          issue.code === "too_big"
            ? tValidation("maxLength", { max: Number(issue.maximum) })
            : tValidation("required");
      }
      setFieldErrors(errors);
      return;
    }
    setFieldErrors({});

    startTransition(async () => {
      const result = await createNote(parsed.data);
      if (result.kind === "error") {
        setFormError(failureMessage(result.error));
        return;
      }
      handleOpenChange(false);
      toast.success(t("created"));
      router.refresh();
    });
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <Plus aria-hidden />
          {t("newNote")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t("newNote")}</DialogTitle>
            <DialogDescription>{t("newNoteDescription")}</DialogDescription>
          </DialogHeader>

          <div className="grid gap-2">
            <Label htmlFor="note-title">{t("titleLabel")}</Label>
            <Input
              aria-describedby={
                fieldErrors.title ? "note-title-error" : undefined
              }
              aria-invalid={fieldErrors.title !== undefined}
              id="note-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
            />
            {fieldErrors.title ? (
              <p className="text-sm text-destructive" id="note-title-error">
                {fieldErrors.title}
              </p>
            ) : null}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="note-body">{t("bodyLabel")}</Label>
            <Textarea
              aria-describedby={
                fieldErrors.body ? "note-body-error" : undefined
              }
              aria-invalid={fieldErrors.body !== undefined}
              className="max-h-64"
              id="note-body"
              value={body}
              onChange={(event) => setBody(event.target.value)}
            />
            {fieldErrors.body ? (
              <p className="text-sm text-destructive" id="note-body-error">
                {fieldErrors.body}
              </p>
            ) : null}
          </div>

          {formError ? (
            <p className="text-sm text-destructive" role="alert">
              {formError}
            </p>
          ) : null}

          <DialogFooter>
            <Button
              disabled={isPending}
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              {tCommon("cancel")}
            </Button>
            <Button aria-busy={isPending} disabled={isPending} type="submit">
              {tCommon("create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

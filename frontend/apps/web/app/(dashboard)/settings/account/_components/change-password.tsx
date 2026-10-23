"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { changePasswordInputSchema } from "@/shared/auth/account";
import { authClient } from "@/shared/auth/client";
import { PASSWORD_LENGTH } from "@/shared/auth/policy";

const CURRENT_ID = "account-settings-current-password";
const CURRENT_ERROR_ID = `${CURRENT_ID}-error`;
const NEW_ID = "account-settings-new-password";
const NEW_HINT_ID = `${NEW_ID}-hint`;

const emptyInput = { currentPassword: "", newPassword: "" };

export function ChangePassword() {
  const t = useTranslations("accountSettings");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [input, setInput] = useState(emptyInput);
  const [currentError, setCurrentError] = useState<string>();
  const [newError, setNewError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();

    const parsed = changePasswordInputSchema.safeParse(input);
    const issues = new Set(parsed.error?.issues.map(({ path }) => path[0]));
    setCurrentError(
      issues.has("currentPassword") ? tValidation("required") : undefined
    );
    setNewError(
      issues.has("newPassword")
        ? t("passwordPolicy", PASSWORD_LENGTH)
        : undefined
    );
    if (!parsed.success) return;

    startTransition(async () => {
      const { error } = await authClient.changePassword({
        ...parsed.data,
        revokeOtherSessions: true,
      });
      if (error) {
        toast.error(
          t(
            error.code === "INVALID_PASSWORD"
              ? "passwordErrors.invalidPassword"
              : "passwordErrors.unexpected"
          )
        );
        return;
      }
      setInput(emptyInput);
      toast.success(t("passwordChanged"));
      router.refresh();
    });
  }

  return (
    <form noValidate className="grid max-w-sm gap-4" onSubmit={handleSubmit}>
      <div className="grid gap-2">
        <Label htmlFor={CURRENT_ID}>{t("currentPasswordLabel")}</Label>
        <Input
          aria-describedby={
            currentError === undefined ? undefined : CURRENT_ERROR_ID
          }
          aria-invalid={currentError !== undefined}
          autoComplete="current-password"
          id={CURRENT_ID}
          type="password"
          value={input.currentPassword}
          onChange={(event) =>
            setInput({ ...input, currentPassword: event.target.value })
          }
        />
        {currentError === undefined ? null : (
          <p className="text-sm text-destructive" id={CURRENT_ERROR_ID}>
            {currentError}
          </p>
        )}
      </div>
      <div className="grid gap-2">
        <Label htmlFor={NEW_ID}>{t("newPasswordLabel")}</Label>
        <Input
          aria-describedby={NEW_HINT_ID}
          aria-invalid={newError !== undefined}
          autoComplete="new-password"
          id={NEW_ID}
          type="password"
          value={input.newPassword}
          onChange={(event) =>
            setInput({ ...input, newPassword: event.target.value })
          }
        />
        <p
          className={
            newError === undefined
              ? "text-sm text-muted-foreground"
              : "text-sm text-destructive"
          }
          id={NEW_HINT_ID}
        >
          {t("passwordPolicy", PASSWORD_LENGTH)}
        </p>
      </div>

      <Button
        aria-busy={isPending}
        className="justify-self-start"
        disabled={isPending}
        type="submit"
      >
        {t("changePassword")}
      </Button>
    </form>
  );
}

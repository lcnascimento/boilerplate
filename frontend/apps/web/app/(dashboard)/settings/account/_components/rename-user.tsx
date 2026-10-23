"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { renameUser } from "@/actions/account";
import { userNameInputSchema } from "@/shared/auth/account";
import { USER_NAME_MAX_LENGTH } from "@/shared/auth/policy";

interface RenameUserProps {
  name: string;
}

const FIELD_ID = "account-settings-name";
const ERROR_ID = `${FIELD_ID}-error`;

export function RenameUser({ name }: RenameUserProps) {
  const t = useTranslations("accountSettings");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [value, setValue] = useState(name);
  const [nameError, setNameError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();

    const parsed = userNameInputSchema.safeParse({ name: value });
    if (!parsed.success) {
      setNameError(
        value.trim() === ""
          ? tValidation("required")
          : tValidation("maxLength", { max: USER_NAME_MAX_LENGTH })
      );
      return;
    }
    setNameError(undefined);

    startTransition(async () => {
      const result = await renameUser(parsed.data);
      if (result.kind === "error") {
        toast.error(t(`renameErrors.${result.code}`));
        return;
      }
      toast.success(t("renamed"));
      router.refresh();
    });
  }

  return (
    <form noValidate className="grid max-w-sm gap-4" onSubmit={handleSubmit}>
      <div className="grid gap-2">
        <Label htmlFor={FIELD_ID}>{t("nameLabel")}</Label>
        <Input
          aria-describedby={nameError === undefined ? undefined : ERROR_ID}
          aria-invalid={nameError !== undefined}
          autoComplete="name"
          id={FIELD_ID}
          type="text"
          value={value}
          onChange={(event) => setValue(event.target.value)}
        />
        {nameError === undefined ? null : (
          <p className="text-sm text-destructive" id={ERROR_ID}>
            {nameError}
          </p>
        )}
      </div>

      <Button
        aria-busy={isPending}
        className="justify-self-start"
        disabled={isPending}
        type="submit"
      >
        {t("save")}
      </Button>
    </form>
  );
}

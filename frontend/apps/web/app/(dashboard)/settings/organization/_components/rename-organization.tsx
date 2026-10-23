"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { renameOrganization } from "@/actions/organization";
import { organizationNameInputSchema } from "@/shared/auth/organizations";
import { ORGANIZATION_NAME_MAX_LENGTH } from "@/shared/auth/policy";

interface RenameOrganizationProps {
  name: string;
  canRename: boolean;
}

const FIELD_ID = "organization-settings-name";
const HINT_ID = `${FIELD_ID}-hint`;

export function RenameOrganization({
  name,
  canRename,
}: RenameOrganizationProps) {
  const t = useTranslations("organizationSettings");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [value, setValue] = useState(name);
  const [nameError, setNameError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();

    const parsed = organizationNameInputSchema.safeParse({ name: value });
    if (!parsed.success) {
      setNameError(
        value.trim() === ""
          ? tValidation("required")
          : tValidation("maxLength", { max: ORGANIZATION_NAME_MAX_LENGTH })
      );
      return;
    }
    setNameError(undefined);

    startTransition(async () => {
      const result = await renameOrganization(parsed.data);
      if (result.kind === "error") {
        toast.error(t(`renameErrors.${result.code}`));
        return;
      }
      toast.success(t("renamed"));
      router.refresh();
    });
  }

  const hint = canRename ? nameError : t("renameNotAllowed");

  return (
    <form noValidate className="grid max-w-sm gap-4" onSubmit={handleSubmit}>
      <div className="grid gap-2">
        <Label htmlFor={FIELD_ID}>{t("nameLabel")}</Label>
        <Input
          aria-describedby={hint === undefined ? undefined : HINT_ID}
          aria-invalid={nameError !== undefined}
          autoComplete="organization"
          id={FIELD_ID}
          readOnly={!canRename}
          type="text"
          value={value}
          onChange={(event) => setValue(event.target.value)}
        />
        {hint === undefined ? null : (
          <p
            className={
              nameError === undefined
                ? "text-sm text-muted-foreground"
                : "text-sm text-destructive"
            }
            id={HINT_ID}
          >
            {hint}
          </p>
        )}
      </div>

      {canRename ? (
        <Button
          aria-busy={isPending}
          className="justify-self-start"
          disabled={isPending}
          type="submit"
        >
          {t("save")}
        </Button>
      ) : null}
    </form>
  );
}

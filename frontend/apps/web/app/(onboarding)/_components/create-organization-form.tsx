"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { createOrganization } from "@/actions/organizations";
import { organizationNameInputSchema } from "@/shared/auth/organizations";
import {
  ORGANIZATION_LIMIT,
  ORGANIZATION_NAME_MAX_LENGTH,
} from "@/shared/auth/policy";
import { SIGNED_IN_HOME } from "@/shared/auth/routes";

const FIELD_ID = "organization-name";
const ERROR_ID = `${FIELD_ID}-error`;

export function CreateOrganizationForm() {
  const t = useTranslations("organizations");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [name, setName] = useState("");
  const [nameError, setNameError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(undefined);

    const parsed = organizationNameInputSchema.safeParse({ name });
    if (!parsed.success) {
      setNameError(
        name.trim() === ""
          ? tValidation("required")
          : tValidation("maxLength", { max: ORGANIZATION_NAME_MAX_LENGTH })
      );
      return;
    }
    setNameError(undefined);

    startTransition(async () => {
      const result = await createOrganization(parsed.data);
      if (result.kind === "error") {
        setFormError(t(`errors.${result.code}`, { limit: ORGANIZATION_LIMIT }));
        return;
      }
      router.replace(SIGNED_IN_HOME);
      router.refresh();
    });
  }

  return (
    <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
      <div className="grid gap-2">
        <Label htmlFor={FIELD_ID}>{t("nameLabel")}</Label>
        <Input
          aria-describedby={nameError ? ERROR_ID : undefined}
          aria-invalid={nameError !== undefined}
          autoComplete="organization"
          id={FIELD_ID}
          type="text"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        {nameError ? (
          <p className="text-sm text-destructive" id={ERROR_ID}>
            {nameError}
          </p>
        ) : null}
      </div>

      {formError ? (
        <p className="text-sm text-destructive" role="alert">
          {formError}
        </p>
      ) : null}

      <Button aria-busy={isPending} disabled={isPending} type="submit">
        {t("create")}
      </Button>
    </form>
  );
}

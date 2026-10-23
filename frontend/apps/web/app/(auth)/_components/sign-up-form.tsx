"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { z } from "zod";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { authClient } from "@/shared/auth/client";
import { PASSWORD_LENGTH } from "@/shared/auth/policy";
import { CREATE_OR_JOIN_HREF } from "@/shared/auth/routes";

import { firstErrorPerField, type FieldErrors } from "./field-errors";

const signUpSchema = z.object({
  name: z.string().trim().min(1),
  email: z.email(),
  password: z.string().min(PASSWORD_LENGTH.min).max(PASSWORD_LENGTH.max),
});

type SignUpInput = z.infer<typeof signUpSchema>;
type Field = keyof SignUpInput;

const fieldSchema = signUpSchema.keyof();

const fields: readonly { field: Field; type: string; autoComplete: string }[] =
  [
    { field: "name", type: "text", autoComplete: "name" },
    { field: "email", type: "email", autoComplete: "email" },
    { field: "password", type: "password", autoComplete: "new-password" },
  ];

const serverFieldErrors: Readonly<Record<string, Field>> = {
  USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL: "email",
  INVALID_EMAIL: "email",
  PASSWORD_TOO_SHORT: "password",
  PASSWORD_TOO_LONG: "password",
};

const emptyInput: Record<Field, string> = {
  name: "",
  email: "",
  password: "",
};

export function SignUpForm() {
  const t = useTranslations("auth");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [input, setInput] = useState(emptyInput);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors<Field>>({});
  const [formError, setFormError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  const labels: Record<Field, string> = {
    name: t("nameLabel"),
    email: t("emailLabel"),
    password: t("passwordLabel"),
  };

  function invalidMessage(field: Field, code: string): string {
    if (input[field].trim() === "") return tValidation("required");
    if (field === "email") {
      return code === "USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL"
        ? t("emailTaken")
        : t("invalidEmail");
    }
    if (field === "password") return t("passwordPolicy", PASSWORD_LENGTH);
    return tValidation("required");
  }

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(undefined);

    const parsed = signUpSchema.safeParse(input);
    if (!parsed.success) {
      setFieldErrors(
        firstErrorPerField(parsed.error, fieldSchema, invalidMessage)
      );
      return;
    }
    setFieldErrors({});

    startTransition(async () => {
      const { error } = await authClient.signUp.email(parsed.data);
      if (error) {
        const code = error.code ?? "";
        const field = serverFieldErrors[code];
        if (field === undefined) setFormError(t("signUpFailed"));
        else setFieldErrors({ [field]: invalidMessage(field, code) });
        return;
      }
      router.replace(CREATE_OR_JOIN_HREF);
      router.refresh();
    });
  }

  return (
    <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
      {fields.map(({ field, type, autoComplete }) => {
        const id = `sign-up-${field}`;
        const errorId = `${id}-error`;
        const fieldError = fieldErrors[field];
        return (
          <div key={field} className="grid gap-2">
            <Label htmlFor={id}>{labels[field]}</Label>
            <Input
              aria-describedby={fieldError ? errorId : undefined}
              aria-invalid={fieldError !== undefined}
              autoComplete={autoComplete}
              id={id}
              type={type}
              value={input[field]}
              onChange={(event) =>
                setInput((current) => ({
                  ...current,
                  [field]: event.target.value,
                }))
              }
            />
            {fieldError ? (
              <p className="text-sm text-destructive" id={errorId}>
                {fieldError}
              </p>
            ) : null}
          </div>
        );
      })}

      {formError ? (
        <p className="text-sm text-destructive" role="alert">
          {formError}
        </p>
      ) : null}

      <Button aria-busy={isPending} disabled={isPending} type="submit">
        {t("signUp")}
      </Button>
    </form>
  );
}

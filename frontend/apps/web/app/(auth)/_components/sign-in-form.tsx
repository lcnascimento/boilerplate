"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { z } from "zod";

import { Button } from "@boilerplate/ui/components/button";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";

import { authClient } from "@/shared/auth/client";

import { firstErrorPerField, type FieldErrors } from "./field-errors";

import type { Route } from "next";

const signInSchema = z.object({
  email: z.email(),
  password: z.string().min(1),
});

type Field = keyof z.infer<typeof signInSchema>;

const fieldSchema = signInSchema.keyof();

const HTTP_UNAUTHORIZED = 401;
const HTTP_TOO_MANY_REQUESTS = 429;

interface SignInFormProps {
  callbackUrl: Route;
}

export function SignInForm({ callbackUrl }: SignInFormProps) {
  const t = useTranslations("auth");
  const tValidation = useTranslations("validation");
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors<Field>>({});
  const [formError, setFormError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function failureMessage(status: number): string {
    if (status === HTTP_UNAUTHORIZED) return t("invalidCredentials");
    if (status === HTTP_TOO_MANY_REQUESTS) return t("tooManyAttempts");
    return t("signInFailed");
  }

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(undefined);

    const parsed = signInSchema.safeParse({ email, password });
    if (!parsed.success) {
      setFieldErrors(
        firstErrorPerField(parsed.error, fieldSchema, (field) =>
          field === "email" && email !== ""
            ? t("invalidEmail")
            : tValidation("required")
        )
      );
      return;
    }
    setFieldErrors({});

    startTransition(async () => {
      const { error } = await authClient.signIn.email(parsed.data);
      if (error) {
        setFormError(failureMessage(error.status));
        return;
      }
      router.replace(callbackUrl);
      router.refresh();
    });
  }

  return (
    <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
      <div className="grid gap-2">
        <Label htmlFor="sign-in-email">{t("emailLabel")}</Label>
        <Input
          aria-describedby={
            fieldErrors.email ? "sign-in-email-error" : undefined
          }
          aria-invalid={fieldErrors.email !== undefined}
          autoComplete="email"
          id="sign-in-email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        {fieldErrors.email ? (
          <p className="text-sm text-destructive" id="sign-in-email-error">
            {fieldErrors.email}
          </p>
        ) : null}
      </div>

      <div className="grid gap-2">
        <Label htmlFor="sign-in-password">{t("passwordLabel")}</Label>
        <Input
          aria-describedby={
            fieldErrors.password ? "sign-in-password-error" : undefined
          }
          aria-invalid={fieldErrors.password !== undefined}
          autoComplete="current-password"
          id="sign-in-password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        {fieldErrors.password ? (
          <p className="text-sm text-destructive" id="sign-in-password-error">
            {fieldErrors.password}
          </p>
        ) : null}
      </div>

      {formError ? (
        <p className="text-sm text-destructive" role="alert">
          {formError}
        </p>
      ) : null}

      <Button aria-busy={isPending} disabled={isPending} type="submit">
        {t("signIn")}
      </Button>
    </form>
  );
}

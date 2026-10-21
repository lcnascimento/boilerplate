import Link from "next/link";
import { useTranslations } from "next-intl";
import { use } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";

import {
  CALLBACK_URL_PARAM,
  parseCallbackUrl,
  SIGN_UP_HREF,
} from "@/shared/auth/routes";

import { SignInForm } from "../_components";

interface SignInPageProps {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

export default function SignInPage({ searchParams }: SignInPageProps) {
  const t = useTranslations("auth");
  const callbackUrl = use(searchParams)[CALLBACK_URL_PARAM];

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h1>{t("signInTitle")}</h1>
        </CardTitle>
        <CardDescription>{t("signInDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <SignInForm
          callbackUrl={parseCallbackUrl(
            typeof callbackUrl === "string" ? callbackUrl : undefined
          )}
        />
      </CardContent>
      <CardFooter className="text-sm text-muted-foreground">
        {t("noAccount")}&nbsp;
        <Link className="underline underline-offset-4" href={SIGN_UP_HREF}>
          {t("goToSignUp")}
        </Link>
      </CardFooter>
    </Card>
  );
}

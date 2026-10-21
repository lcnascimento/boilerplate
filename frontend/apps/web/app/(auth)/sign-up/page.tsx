import Link from "next/link";
import { useTranslations } from "next-intl";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";

import { SIGN_IN_HREF } from "@/shared/auth/routes";

import { SignUpForm } from "../_components";

export default function SignUpPage() {
  const t = useTranslations("auth");

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h1>{t("signUpTitle")}</h1>
        </CardTitle>
        <CardDescription>{t("signUpDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <SignUpForm />
      </CardContent>
      <CardFooter className="text-sm text-muted-foreground">
        {t("haveAccount")}&nbsp;
        <Link className="underline underline-offset-4" href={SIGN_IN_HREF}>
          {t("goToSignIn")}
        </Link>
      </CardFooter>
    </Card>
  );
}

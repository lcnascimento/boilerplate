import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";

import { Button } from "@boilerplate/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@boilerplate/ui/components/card";

import { listReceivedInvitations } from "@/actions/invitations";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF, SIGNED_IN_HOME } from "@/shared/auth/routes";
import { getRequestSession } from "@/shared/auth/session";
import { SignOutButton } from "@/shared/components";

import { CreateOrganizationForm, ReceivedInvitations } from "../../_components";

export default async function CreateOrJoinPage() {
  const [session, requestActor, invitations, t] = await Promise.all([
    getRequestSession(),
    getRequestActor(),
    listReceivedInvitations(),
    getTranslations("organizations"),
  ]);
  if (session === null || requestActor.kind === "signed-out") {
    redirect(SIGN_IN_HREF);
  }
  const active =
    requestActor.kind === "actor" ? requestActor.organizations.active : null;

  return (
    <main className="flex min-h-svh flex-col">
      <header className="flex items-center justify-between gap-4 p-4">
        {active === null ? (
          <span />
        ) : (
          <Button asChild size="sm" variant="ghost">
            <Link href={SIGNED_IN_HOME}>
              <ArrowLeft aria-hidden />
              {t("backTo", { organization: active.name })}
            </Link>
          </Button>
        )}
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <span className="hidden sm:inline">
            {t("signedInAs", { email: session.user.email })}
          </span>
          <SignOutButton />
        </div>
      </header>

      <div className="flex flex-1 items-center justify-center p-6">
        <div className="grid w-full max-w-md gap-6">
          <div className="grid gap-2 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">
              {t("createOrJoin")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("createOrJoinDescription")}
            </p>
          </div>

          <ReceivedInvitations invitations={invitations} />

          <Card>
            <CardHeader>
              <CardTitle>
                <h2>{t("createTitle")}</h2>
              </CardTitle>
              <CardDescription>{t("createDescription")}</CardDescription>
            </CardHeader>
            <CardContent>
              <CreateOrganizationForm />
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  );
}

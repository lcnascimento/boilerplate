"use client";

import { CircleAlert, Unplug } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";

import { Button } from "@boilerplate/ui/components/button";
import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

import type { APIError } from "@boilerplate/api/core";

interface NotesErrorProps {
  error: Pick<APIError, "code">;
}

/** A failed read: says whether the service is down and offers a retry. */
export function NotesError({ error }: NotesErrorProps) {
  const t = useTranslations("notes");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const unavailable = error.code === "UNAVAILABLE";

  return (
    <EmptyState
      icon={unavailable ? Unplug : CircleAlert}
      role="alert"
      title={unavailable ? t("unavailableTitle") : t("loadFailedTitle")}
      action={
        <Button
          aria-busy={isPending}
          disabled={isPending}
          variant="outline"
          onClick={() => startTransition(() => router.refresh())}
        >
          {tCommon("tryAgain")}
        </Button>
      }
      description={
        <>
          {unavailable
            ? t("unavailableDescription")
            : t("loadFailedDescription")}
          <code className="mt-2 block text-xs">{error.code}</code>
        </>
      }
    />
  );
}

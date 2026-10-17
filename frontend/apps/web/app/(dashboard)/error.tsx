"use client";

import { CircleAlert } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";
import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

interface DashboardErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function DashboardError({ error, reset }: DashboardErrorProps) {
  const t = useTranslations("errors");
  const tCommon = useTranslations("common");

  return (
    <EmptyState
      action={<Button onClick={reset}>{tCommon("tryAgain")}</Button>}
      icon={CircleAlert}
      role="alert"
      title={t("generic")}
      description={
        <>
          {t("retryHint")}
          {error.digest ? (
            <code className="mt-2 block text-xs">{error.digest}</code>
          ) : null}
        </>
      }
    />
  );
}

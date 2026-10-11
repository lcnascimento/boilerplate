import { FileQuestion } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";

import { Button } from "@boilerplate/ui/components/button";
import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

export default function DashboardNotFound() {
  const t = useTranslations("errors");
  const tNav = useTranslations("nav");

  return (
    <EmptyState
      icon={FileQuestion}
      title={t("notFound")}
      action={
        <Button asChild variant="outline">
          <Link href="/">{tNav("home")}</Link>
        </Button>
      }
    />
  );
}

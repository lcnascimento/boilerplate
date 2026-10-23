"use client";

import { X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";

import { cancelInvitation } from "@/actions/invitations";

import type { SentInvitation } from "@/shared/auth/invitations";

interface CancelInvitationButtonProps {
  invitation: Pick<SentInvitation, "id" | "email">;
}

export function CancelInvitationButton({
  invitation,
}: CancelInvitationButtonProps) {
  const t = useTranslations("invitations");
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  function handleClick() {
    startTransition(async () => {
      const result = await cancelInvitation({ invitationId: invitation.id });
      if (result.kind === "error") {
        toast.error(t(`errors.${result.code}`));
        return;
      }
      toast.success(t("canceled"));
      router.refresh();
    });
  }

  return (
    <Button
      aria-busy={isPending}
      aria-label={t("cancelInvitationOf", { email: invitation.email })}
      disabled={isPending}
      size="icon"
      variant="ghost"
      onClick={handleClick}
    >
      <X aria-hidden />
    </Button>
  );
}

"use client";

import { UserPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useState, useTransition, type SubmitEvent } from "react";
import { toast } from "sonner";

import { Button } from "@boilerplate/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@boilerplate/ui/components/dialog";
import { Input } from "@boilerplate/ui/components/input";
import { Label } from "@boilerplate/ui/components/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@boilerplate/ui/components/select";

import { createInvitation } from "@/actions/invitations";
import {
  createInvitationInputSchema,
  invitableRoleSchema,
  invitableRoles,
  type InvitableRole,
} from "@/shared/auth/invitations";
import { INVITATION_EXPIRES_IN_DAYS } from "@/shared/auth/policy";

const DEFAULT_ROLE: InvitableRole = "viewer";

export function InviteMemberDialog() {
  const t = useTranslations("invitations");
  const tMembers = useTranslations("members");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<InvitableRole>(DEFAULT_ROLE);
  const [emailError, setEmailError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [isPending, startTransition] = useTransition();

  function handleOpenChange(next: boolean) {
    if (!next) {
      setEmail("");
      setRole(DEFAULT_ROLE);
      setEmailError(undefined);
      setFormError(undefined);
    }
    setOpen(next);
  }

  function handleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(undefined);

    const parsed = createInvitationInputSchema.safeParse({
      email: email.trim(),
      role,
    });
    if (!parsed.success) {
      setEmailError(t("invalidEmail"));
      return;
    }
    setEmailError(undefined);

    startTransition(async () => {
      const result = await createInvitation(parsed.data);
      if (result.kind === "error") {
        setFormError(t(`errors.${result.code}`));
        return;
      }
      handleOpenChange(false);
      toast.success(t("sent"));
      router.refresh();
    });
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <UserPlus aria-hidden />
          {t("invite")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t("inviteTitle")}</DialogTitle>
            <DialogDescription>
              {t("inviteDescription", { days: INVITATION_EXPIRES_IN_DAYS })}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-2">
            <Label htmlFor="invitation-email">{t("emailLabel")}</Label>
            <Input
              aria-describedby={
                emailError ? "invitation-email-error" : undefined
              }
              aria-invalid={emailError !== undefined}
              autoComplete="off"
              id="invitation-email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
            {emailError ? (
              <p
                className="text-sm text-destructive"
                id="invitation-email-error"
              >
                {emailError}
              </p>
            ) : null}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="invitation-role">{t("roleLabel")}</Label>
            <Select
              value={role}
              onValueChange={(value) =>
                setRole(invitableRoleSchema.parse(value))
              }
            >
              <SelectTrigger className="w-full" id="invitation-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {invitableRoles.map((option) => (
                  <SelectItem key={option} value={option}>
                    {tMembers(`roles.${option}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {formError ? (
            <p className="text-sm text-destructive" role="alert">
              {formError}
            </p>
          ) : null}

          <DialogFooter>
            <Button
              disabled={isPending}
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              {tCommon("cancel")}
            </Button>
            <Button aria-busy={isPending} disabled={isPending} type="submit">
              {t("send")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

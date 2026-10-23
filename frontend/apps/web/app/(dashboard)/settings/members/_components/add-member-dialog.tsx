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

import { addMember } from "@/actions/members";
import {
  addMemberInputSchema,
  memberRoleSchema,
  type MemberRole,
} from "@/shared/auth/members";

interface AddMemberDialogProps {
  assignableRoles: MemberRole[];
}

const DEFAULT_ROLE: MemberRole = "viewer";

export function AddMemberDialog({ assignableRoles }: AddMemberDialogProps) {
  const t = useTranslations("members");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<MemberRole>(DEFAULT_ROLE);
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

    const parsed = addMemberInputSchema.safeParse({
      email: email.trim(),
      role,
    });
    if (!parsed.success) {
      setEmailError(t("invalidEmail"));
      return;
    }
    setEmailError(undefined);

    startTransition(async () => {
      const result = await addMember(parsed.data);
      if (result.kind === "error") {
        setFormError(t(`errors.${result.code}`));
        return;
      }
      handleOpenChange(false);
      toast.success(t("added"));
      router.refresh();
    });
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <UserPlus aria-hidden />
          {t("addMember")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form noValidate className="grid gap-4" onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t("addMember")}</DialogTitle>
            <DialogDescription>{t("addMemberDescription")}</DialogDescription>
          </DialogHeader>

          <div className="grid gap-2">
            <Label htmlFor="member-email">{t("emailLabel")}</Label>
            <Input
              aria-describedby={emailError ? "member-email-error" : undefined}
              aria-invalid={emailError !== undefined}
              autoComplete="off"
              id="member-email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
            {emailError ? (
              <p className="text-sm text-destructive" id="member-email-error">
                {emailError}
              </p>
            ) : null}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="member-role">{t("roleLabel")}</Label>
            <Select
              value={role}
              onValueChange={(value) => setRole(memberRoleSchema.parse(value))}
            >
              <SelectTrigger className="w-full" id="member-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {assignableRoles.map((option) => (
                  <SelectItem key={option} value={option}>
                    {t(`roles.${option}`)}
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
              {t("add")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

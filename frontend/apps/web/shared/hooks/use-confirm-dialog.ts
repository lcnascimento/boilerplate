"use client";

import { useState, useTransition, type MouseEvent } from "react";

export function useConfirmDialog(confirm: () => Promise<"done" | "failed">) {
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();

  function handleConfirm(event: MouseEvent<HTMLButtonElement>) {
    // Radix closes the dialog on the action's click.
    event.preventDefault();

    startTransition(async () => {
      if ((await confirm()) === "done") setOpen(false);
    });
  }

  return { open, setOpen, isPending, handleConfirm };
}

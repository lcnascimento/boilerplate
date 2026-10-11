import { expect, fn, screen, waitFor } from "storybook/test";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "./alert-dialog";
import { Button } from "./button";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/AlertDialog",
  args: { onConfirm: fn() },
  render: ({ onConfirm }) => (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="destructive">Excluir nota</Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Excluir esta nota?</AlertDialogTitle>
          <AlertDialogDescription>
            Esta ação não pode ser desfeita.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>Excluir</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  ),
} satisfies Meta<{ onConfirm: () => void }>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Confirm: Story = {
  play: async ({ args, canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Excluir nota" }));
    const dialog = await screen.findByRole("alertdialog", {
      name: "Excluir esta nota?",
    });
    await waitFor(() => expect(dialog).toBeVisible());

    await userEvent.click(screen.getByRole("button", { name: "Excluir" }));
    await expect(args.onConfirm).toHaveBeenCalledOnce();
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
  },
};

export const Cancel: Story = {
  play: async ({ args, canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Excluir nota" }));
    await userEvent.click(
      await screen.findByRole("button", { name: "Cancelar" })
    );
    await expect(args.onConfirm).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
  },
};

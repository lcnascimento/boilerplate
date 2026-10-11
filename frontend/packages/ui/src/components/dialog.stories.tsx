import { expect, screen, waitFor } from "storybook/test";

import { Button } from "./button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./dialog";
import { Input } from "./input";
import { Label } from "./label";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Dialog",
  component: Dialog,
  render: () => (
    <Dialog>
      <DialogTrigger asChild>
        <Button>Nova nota</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Nova nota</DialogTitle>
          <DialogDescription>Dê um título para a nota.</DialogDescription>
        </DialogHeader>
        <div className="grid gap-2">
          <Label htmlFor="note-title">Título</Label>
          <Input id="note-title" />
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancelar</Button>
          </DialogClose>
          <Button>Criar</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
} satisfies Meta<typeof Dialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const OpenAndClose: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Nova nota" }));
    const dialog = await screen.findByRole("dialog", { name: "Nova nota" });
    await waitFor(() => expect(dialog).toBeVisible());
    await expect(screen.getByLabelText("Título")).toHaveFocus();

    await userEvent.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  },
};

export const CloseWithEscape: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Nova nota" }));
    await screen.findByRole("dialog", { name: "Nova nota" });

    await userEvent.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  },
};

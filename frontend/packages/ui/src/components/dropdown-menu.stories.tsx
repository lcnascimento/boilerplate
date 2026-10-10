import { expect, fn, screen, waitFor } from "storybook/test";

import { Button } from "./button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./dropdown-menu";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/DropdownMenu",
  args: { onSelect: fn() },
  render: ({ onSelect }) => (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline">Usuário local</Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuLabel>Minha conta</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={onSelect}>Preferências</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ),
} satisfies Meta<{ onSelect: () => void }>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SelectItem: Story = {
  play: async ({ args, canvas, userEvent }) => {
    await userEvent.click(
      canvas.getByRole("button", { name: "Usuário local" })
    );
    const item = await screen.findByRole("menuitem", { name: "Preferências" });

    await userEvent.click(item);
    await expect(args.onSelect).toHaveBeenCalledOnce();
    await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
  },
};

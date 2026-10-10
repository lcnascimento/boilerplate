import { NotebookPenIcon, SearchIcon } from "lucide-react";
import { expect, fn } from "storybook/test";

import { Button } from "../components/button";

import { EmptyState } from "./empty-state";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Primitives/EmptyState",
  component: EmptyState,
  args: {
    icon: NotebookPenIcon,
    title: "Nenhuma nota ainda",
    description: "Crie a primeira nota para começar.",
  },
} satisfies Meta<typeof EmptyState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("heading", { level: 2, name: "Nenhuma nota ainda" })
    ).toBeVisible();
  },
};

const onCreate = fn();

export const WithAction: Story = {
  args: { action: <Button onClick={onCreate}>Nova nota</Button> },
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Nova nota" }));
    await expect(onCreate).toHaveBeenCalledOnce();
  },
};

export const WithoutIcon: Story = {
  args: {
    icon: undefined,
    title: "Nenhum resultado",
    description: undefined,
  },
};

export const SearchIconVariant: Story = {
  args: { icon: SearchIcon, title: "Nenhum resultado" },
};

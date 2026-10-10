import { PlusIcon } from "lucide-react";
import { expect } from "storybook/test";

import { Button } from "../components/button";

import { PageHeader } from "./page-header";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Primitives/PageHeader",
  component: PageHeader,
  parameters: { layout: "padded" },
  args: {
    title: "Notas",
    description: "Suas anotações, da mais nova para a mais antiga.",
  },
} satisfies Meta<typeof PageHeader>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("heading", { level: 1, name: "Notas" })
    ).toBeVisible();
    await expect(
      canvas.getByText("Suas anotações, da mais nova para a mais antiga.")
    ).toBeVisible();
  },
};

export const WithAction: Story = {
  args: {
    action: (
      <Button>
        <PlusIcon /> Nova nota
      </Button>
    ),
  },
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("button", { name: "Nova nota" })
    ).toBeVisible();
  },
};

export const TitleOnly: Story = {
  args: { title: "Bem-vindo", description: undefined },
};

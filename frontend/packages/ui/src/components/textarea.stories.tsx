import { expect } from "storybook/test";

import { Textarea } from "./textarea";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Textarea",
  component: Textarea,
  args: { "aria-label": "Corpo", placeholder: "Escreva a nota" },
} satisfies Meta<typeof Textarea>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas, userEvent }) => {
    const textarea = canvas.getByRole("textbox", { name: "Corpo" });
    await userEvent.type(textarea, "Pão{enter}Leite");
    await expect(textarea).toHaveValue("Pão\nLeite");
  },
};

export const Disabled: Story = {
  args: { disabled: true },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("textbox", { name: "Corpo" })).toBeDisabled();
  },
};

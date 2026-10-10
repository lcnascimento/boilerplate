import { expect } from "storybook/test";

import { Input } from "./input";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Input",
  component: Input,
  args: { "aria-label": "Título", placeholder: "Lista de compras" },
} satisfies Meta<typeof Input>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas, userEvent }) => {
    const input = canvas.getByRole("textbox", { name: "Título" });
    await userEvent.type(input, "Ideias");
    await expect(input).toHaveValue("Ideias");
  },
};

export const Invalid: Story = {
  args: { "aria-invalid": true, defaultValue: "" },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("textbox", { name: "Título" })).toBeInvalid();
  },
};

export const Disabled: Story = {
  args: { disabled: true },
  play: async ({ canvas }) => {
    await expect(
      canvas.getByRole("textbox", { name: "Título" })
    ).toBeDisabled();
  },
};

import { expect } from "storybook/test";

import { Input } from "./input";
import { Label } from "./label";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Label",
  component: Label,
  render: () => (
    <div className="grid gap-2">
      <Label htmlFor="title">Título</Label>
      <Input id="title" />
    </div>
  ),
} satisfies Meta<typeof Label>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LabelsItsField: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByText("Título"));
    await expect(canvas.getByRole("textbox", { name: "Título" })).toHaveFocus();
  },
};

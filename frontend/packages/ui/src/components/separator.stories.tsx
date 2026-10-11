import { expect } from "storybook/test";

import { Separator } from "./separator";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Separator",
  component: Separator,
  args: { orientation: "horizontal", decorative: false },
  render: (args) => (
    <div
      className={
        args.orientation === "vertical"
          ? "flex h-6 items-center gap-4"
          : "flex w-48 flex-col gap-4"
      }
    >
      <span>Notas</span>
      <Separator {...args} />
      <span>Preferências</span>
    </div>
  ),
} satisfies Meta<typeof Separator>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Horizontal: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("separator")).toBeInTheDocument();
  },
};

export const Vertical: Story = {
  args: { orientation: "vertical" },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("separator")).toHaveAttribute(
      "aria-orientation",
      "vertical"
    );
  },
};

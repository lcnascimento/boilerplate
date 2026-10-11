import { expect } from "storybook/test";

import { Avatar, AvatarFallback, AvatarImage } from "./avatar";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Avatar",
  component: Avatar,
  args: { size: "default" },
  argTypes: {
    size: { control: "select", options: ["sm", "default", "lg"] },
  },
  render: (args) => (
    <Avatar {...args}>
      <AvatarImage alt="Usuário local" />
      <AvatarFallback>UL</AvatarFallback>
    </Avatar>
  ),
} satisfies Meta<typeof Avatar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Fallback: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByText("UL")).toBeVisible();
  },
};

export const Large: Story = {
  args: { size: "lg" },
};

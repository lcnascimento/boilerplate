import { toast } from "sonner";
import { expect, screen, waitFor } from "storybook/test";

import { Button } from "./button";
import { Toaster } from "./sonner";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Sonner",
  component: Toaster,
  render: (args) => (
    <>
      <Toaster {...args} />
      <Button onClick={() => toast.success("Nota criada")}>Criar nota</Button>
    </>
  ),
} satisfies Meta<typeof Toaster>;

export default meta;
type Story = StoryObj<typeof meta>;

export const ShowToast: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Criar nota" }));
    const toast = await screen.findByText("Nota criada");
    await waitFor(() => expect(toast).toBeVisible());
  },
};

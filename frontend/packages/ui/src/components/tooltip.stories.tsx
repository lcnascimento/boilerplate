import { expect, screen } from "storybook/test";

import { Button } from "./button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Tooltip",
  component: Tooltip,
  decorators: [
    (Story) => (
      <TooltipProvider>
        <Story />
      </TooltipProvider>
    ),
  ],
  render: () => (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="outline">Notas</Button>
      </TooltipTrigger>
      <TooltipContent>Abrir suas notas</TooltipContent>
    </Tooltip>
  ),
} satisfies Meta<typeof Tooltip>;

export default meta;
type Story = StoryObj<typeof meta>;

export const ShowOnHover: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.hover(canvas.getByRole("button", { name: "Notas" }));
    await expect(await screen.findByRole("tooltip")).toHaveTextContent(
      "Abrir suas notas"
    );
  },
};

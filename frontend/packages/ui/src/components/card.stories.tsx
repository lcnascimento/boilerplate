import { expect } from "storybook/test";

import { Button } from "./button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./card";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Card",
  component: Card,
  render: (args) => (
    <Card className="w-80" {...args}>
      <CardHeader>
        <CardTitle>Lista de compras</CardTitle>
        <CardDescription>Criada há 2 horas</CardDescription>
        <CardAction>
          <Button size="sm" variant="ghost">
            Editar
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>Pão, leite e café.</CardContent>
      <CardFooter>
        <Button variant="outline">Abrir</Button>
      </CardFooter>
    </Card>
  ),
} satisfies Meta<typeof Card>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByText("Lista de compras")).toBeVisible();
    await expect(canvas.getByText("Pão, leite e café.")).toBeVisible();
    await expect(canvas.getByRole("button", { name: "Abrir" })).toBeVisible();
  },
};

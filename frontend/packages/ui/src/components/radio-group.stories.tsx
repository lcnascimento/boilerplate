import { expect, fn } from "storybook/test";

import { Label } from "./label";
import { RadioGroup, RadioGroupItem } from "./radio-group";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/RadioGroup",
  component: RadioGroup,
  args: { defaultValue: "system", onValueChange: fn(), "aria-label": "Tema" },
  render: (args) => (
    <RadioGroup {...args}>
      {[
        ["light", "Claro"],
        ["dark", "Escuro"],
        ["system", "Sistema"],
      ].map(([value, label]) => (
        <div key={value} className="flex items-center gap-2">
          <RadioGroupItem id={`theme-${String(value)}`} value={value ?? ""} />
          <Label htmlFor={`theme-${String(value)}`}>{label}</Label>
        </div>
      ))}
    </RadioGroup>
  ),
} satisfies Meta<typeof RadioGroup>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SelectOption: Story = {
  play: async ({ args, canvas, userEvent }) => {
    await expect(canvas.getByRole("radio", { name: "Sistema" })).toBeChecked();

    await userEvent.click(canvas.getByRole("radio", { name: "Escuro" }));
    await expect(canvas.getByRole("radio", { name: "Escuro" })).toBeChecked();
    await expect(args.onValueChange).toHaveBeenCalledWith("dark");
  },
};

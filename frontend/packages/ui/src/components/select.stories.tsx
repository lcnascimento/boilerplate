import { expect, fn, screen, waitFor } from "storybook/test";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./select";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Select",
  component: Select,
  args: { defaultValue: "viewer", onValueChange: fn() },
  render: (args) => (
    <Select {...args}>
      <SelectTrigger aria-label="Role" className="w-40">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="admin">Admin</SelectItem>
        <SelectItem value="viewer">Viewer</SelectItem>
      </SelectContent>
    </Select>
  ),
} satisfies Meta<typeof Select>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SelectOption: Story = {
  play: async ({ args, canvas, userEvent }) => {
    const trigger = canvas.getByRole("combobox", { name: "Role" });
    await expect(trigger).toHaveTextContent("Viewer");

    await userEvent.click(trigger);
    await userEvent.click(await screen.findByRole("option", { name: "Admin" }));

    await expect(args.onValueChange).toHaveBeenCalledWith("admin");
    await waitFor(() => expect(trigger).toHaveTextContent("Admin"));
    await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
  },
};

export const Disabled: Story = {
  args: { disabled: true },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("combobox", { name: "Role" })).toBeDisabled();
  },
};

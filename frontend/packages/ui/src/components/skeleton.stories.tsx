import { Skeleton } from "./skeleton";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Skeleton",
  component: Skeleton,
  render: () => (
    <div className="flex items-center gap-4">
      <Skeleton className="size-12 rounded-full" />
      <div className="grid gap-2">
        <Skeleton className="h-4 w-56" />
        <Skeleton className="h-4 w-40" />
      </div>
    </div>
  ),
} satisfies Meta<typeof Skeleton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

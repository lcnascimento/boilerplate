import { HouseIcon, NotebookPenIcon } from "lucide-react";
import { expect } from "storybook/test";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarProvider,
  SidebarTrigger,
} from "./sidebar";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Sidebar",
  component: SidebarProvider,
  parameters: { layout: "fullscreen" },
  args: { defaultOpen: true },
  render: (args) => (
    <SidebarProvider {...args}>
      <Sidebar collapsible="icon">
        <SidebarHeader>Boilerplate</SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Menu</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton isActive tooltip="Início">
                    <HouseIcon />
                    <span>Início</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton tooltip="Notas">
                    <NotebookPenIcon />
                    <span>Notas</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuSkeleton showIcon />
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-12 items-center border-b px-2">
          <SidebarTrigger />
        </header>
        <div className="p-4">Conteúdo</div>
      </SidebarInset>
    </SidebarProvider>
  ),
} satisfies Meta<typeof SidebarProvider>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Expanded: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByRole("button", { name: "Notas" })).toBeVisible();
  },
};

export const ToggleCollapsed: Story = {
  play: async ({ canvas, canvasElement, userEvent }) => {
    const sidebar = canvasElement.querySelector('[data-slot="sidebar"]');
    await expect(sidebar).toHaveAttribute("data-state", "expanded");

    await userEvent.click(
      canvas.getByRole("button", { name: "Toggle Sidebar" })
    );
    await expect(sidebar).toHaveAttribute("data-state", "collapsed");
  },
};

export const Collapsed: Story = {
  args: { defaultOpen: false },
};

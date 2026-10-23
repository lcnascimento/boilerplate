import { expect, within } from "storybook/test";

import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./table";

import type { Meta, StoryObj } from "@storybook/react-vite";

const meta = {
  title: "Components/Table",
  component: Table,
  render: (args) => (
    <Table {...args}>
      <TableCaption>Members da Acme</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>Nome</TableHead>
          <TableHead>Email</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow>
          <TableCell>Ana Owner</TableCell>
          <TableCell>owner@acme.test</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Bruno Admin</TableCell>
          <TableCell>admin@acme.test</TableCell>
        </TableRow>
      </TableBody>
    </Table>
  ),
} satisfies Meta<typeof Table>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  play: async ({ canvas }) => {
    const table = canvas.getByRole("table", { name: "Members da Acme" });
    await expect(
      within(table)
        .getAllByRole("columnheader")
        .map((th) => th.textContent)
    ).toEqual(["Nome", "Email"]);

    const [, firstRow] = within(table).getAllByRole("row");
    await expect(
      within(firstRow ?? table)
        .getAllByRole("cell")
        .map((td) => td.textContent)
    ).toEqual(["Ana Owner", "owner@acme.test"]);
  },
};

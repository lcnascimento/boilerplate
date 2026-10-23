/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { deleteOrganization } from "@/actions/organization";

import {
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { DeleteOrganization } from "./delete-organization";

const push = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh }) }));
vi.mock("@/actions/organization", () => ({ deleteOrganization: vi.fn() }));

const deleteOrganizationMock = vi.mocked(deleteOrganization);

beforeEach(() => {
  stubBrowserApis();
  deleteOrganizationMock.mockReset();
  push.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

async function openDialog(user: ReturnType<typeof renderWithMessages>) {
  await user.click(
    screen.getByRole("button", { name: "Excluir Organization" })
  );
  const dialog = within(
    screen.getByRole("alertdialog", { name: "Excluir Acme?" })
  );
  return {
    confirmation: dialog.getByRole("textbox", {
      name: "Digite Acme para confirmar",
    }),
    confirm: dialog.getByRole("button", { name: "Excluir" }),
  };
}

describe("DeleteOrganization", () => {
  it("deletes only after the exact name is typed, then goes home", async () => {
    deleteOrganizationMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(<DeleteOrganization name="Acme" />);

    const { confirmation, confirm } = await openDialog(user);
    expect(confirm.hasAttribute("disabled")).toBe(true);
    await user.type(confirmation, "acme");
    expect(confirm.hasAttribute("disabled")).toBe(true);
    await user.clear(confirmation);
    await user.type(confirmation, "Acme");
    expect(confirm.hasAttribute("disabled")).toBe(false);
    await user.click(confirm);

    expect(deleteOrganizationMock).toHaveBeenCalledWith({
      confirmation: "Acme",
    });
    expect(await screen.findByText("Organization Acme excluída.")).toBeTruthy();
    expect(push).toHaveBeenCalledWith("/");
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains a refused deletion and stays on the page", async () => {
    deleteOrganizationMock.mockResolvedValue({
      kind: "error",
      code: "forbidden",
    });
    const user = renderWithMessages(<DeleteOrganization name="Acme" />);

    const { confirmation, confirm } = await openDialog(user);
    await user.type(confirmation, "Acme");
    await user.click(confirm);

    expect(
      await screen.findByText("Só o Owner pode excluir a Organization.")
    ).toBeTruthy();
    expect(push).not.toHaveBeenCalled();
  });
});

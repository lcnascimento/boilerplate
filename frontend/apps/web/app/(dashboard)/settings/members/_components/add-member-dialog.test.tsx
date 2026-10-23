/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { addMember } from "@/actions/members";
import { memberPolicy, type Member } from "@/shared/auth/members";

import { AddMemberDialog } from "./add-member-dialog";
import {
  acme,
  ana,
  bruno,
  renderWithMessages,
  stubBrowserApis,
} from "./test-utils";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/members", () => ({ addMember: vi.fn() }));

const addMemberMock = vi.mocked(addMember);

async function openDialog(actor: Member) {
  const user = renderWithMessages(
    <AddMemberDialog
      assignableRoles={memberPolicy(actor, acme).assignableRoles}
    />
  );
  await user.click(screen.getByRole("button", { name: "Adicionar Member" }));
  const dialog = screen.getByRole("dialog", { name: "Adicionar Member" });
  return { user, dialog: within(dialog) };
}

beforeEach(() => {
  stubBrowserApis();
  addMemberMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("AddMemberDialog", () => {
  for (const actor of [ana, bruno]) {
    it(`offers ${actor.name} no Owner Role`, async () => {
      const { user, dialog } = await openDialog(actor);

      await user.click(dialog.getByRole("combobox", { name: "Role" }));

      expect(
        screen.getAllByRole("option").map((option) => option.textContent)
      ).toEqual(["Admin", "Viewer"]);
    });
  }

  it("asks for a valid email before sending anything", async () => {
    const { user, dialog } = await openDialog(ana);

    await user.type(dialog.getByLabelText("Email"), "diego");
    await user.click(dialog.getByRole("button", { name: "Adicionar" }));

    expect(dialog.getByText("Informe um email válido.")).toBeTruthy();
    expect(addMemberMock).not.toHaveBeenCalled();
  });

  it("adds the User as Viewer by default, closes and refreshes", async () => {
    addMemberMock.mockResolvedValue({ kind: "ok" });
    const { user, dialog } = await openDialog(ana);

    await user.type(dialog.getByLabelText("Email"), " owner@globex.test ");
    await user.click(dialog.getByRole("button", { name: "Adicionar" }));

    expect(addMemberMock).toHaveBeenCalledWith({
      email: "owner@globex.test",
      role: "viewer",
    });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it.each([
    ["userNotFound", "Nenhum User cadastrado com esse email."],
    ["alreadyMember", "Esse User já é Member desta Organization."],
  ] as const)("explains %s and stays open", async (code, message) => {
    addMemberMock.mockResolvedValue({ kind: "error", code });
    const { user, dialog } = await openDialog(ana);

    await user.type(dialog.getByLabelText("Email"), "someone@acme.test");
    await user.click(dialog.getByRole("button", { name: "Adicionar" }));

    expect(dialog.getByRole("alert").textContent).toBe(message);
    expect(refresh).not.toHaveBeenCalled();
  });
});

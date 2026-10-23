/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createInvitation } from "@/actions/invitations";

import { InviteMemberDialog } from "./invite-member-dialog";
import { renderWithMessages, stubBrowserApis } from "./test-utils";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/invitations", () => ({ createInvitation: vi.fn() }));

const createInvitationMock = vi.mocked(createInvitation);

async function openDialog() {
  const user = renderWithMessages(<InviteMemberDialog />);
  await user.click(screen.getByRole("button", { name: "Convidar" }));
  const dialog = screen.getByRole("dialog", {
    name: "Convidar para a Organization",
  });
  return { user, dialog: within(dialog) };
}

beforeEach(() => {
  stubBrowserApis();
  createInvitationMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("InviteMemberDialog", () => {
  it("offers no Owner Role and says how long the invitation lasts", async () => {
    const { user, dialog } = await openDialog();

    expect(
      dialog.getByText(
        'A pessoa vê o convite em "Criar ou entrar" ao entrar com este email, mesmo que ainda não tenha cadastro. O convite vale 7 dias.'
      )
    ).toBeTruthy();
    await user.click(dialog.getByRole("combobox", { name: "Role" }));
    expect(
      screen.getAllByRole("option").map((option) => option.textContent)
    ).toEqual(["Admin", "Viewer"]);
  });

  it("asks for a valid email before sending anything", async () => {
    const { user, dialog } = await openDialog();

    await user.type(dialog.getByLabelText("Email"), "diego");
    await user.click(dialog.getByRole("button", { name: "Convidar" }));

    expect(dialog.getByText("Informe um email válido.")).toBeTruthy();
    expect(createInvitationMock).not.toHaveBeenCalled();
  });

  it("invites the email as Viewer by default, closes and refreshes", async () => {
    createInvitationMock.mockResolvedValue({ kind: "ok" });
    const { user, dialog } = await openDialog();

    await user.type(dialog.getByLabelText("Email"), " new@globex.test ");
    await user.click(dialog.getByRole("button", { name: "Convidar" }));

    expect(createInvitationMock).toHaveBeenCalledWith({
      email: "new@globex.test",
      role: "viewer",
    });
    expect(await screen.findByText("Convite criado.")).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it.each([
    ["alreadyMember", "Esse email já é de um Member desta Organization."],
    [
      "alreadyInvited",
      "Esse email já tem um convite pendente para esta Organization.",
    ],
  ] as const)("explains %s and stays open", async (code, message) => {
    createInvitationMock.mockResolvedValue({ kind: "error", code });
    const { user, dialog } = await openDialog();

    await user.type(dialog.getByLabelText("Email"), "someone@acme.test");
    await user.click(dialog.getByRole("button", { name: "Convidar" }));

    expect(dialog.getByRole("alert").textContent).toBe(message);
    expect(refresh).not.toHaveBeenCalled();
  });
});

/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { deleteAccount } from "@/actions/account";
import { authClient } from "@/shared/auth/client";

import {
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { DeleteAccount } from "./delete-account";

const push = vi.fn();
const replace = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, replace, refresh }),
}));
vi.mock("@/actions/account", () => ({ deleteAccount: vi.fn() }));
vi.mock("@/shared/auth/client", () => ({
  authClient: { organization: { setActive: vi.fn() } },
}));

const deleteAccountMock = vi.mocked(deleteAccount);
const setActive = vi.mocked(authClient.organization.setActive);

beforeEach(() => {
  stubBrowserApis();
  deleteAccountMock.mockReset();
  setActive.mockReset();
  push.mockReset();
  replace.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

async function openDialog(user: ReturnType<typeof renderWithMessages>) {
  await user.click(screen.getByRole("button", { name: "Excluir conta" }));
  const dialog = within(
    screen.getByRole("alertdialog", { name: "Excluir a sua conta?" })
  );
  return {
    password: dialog.getByLabelText("Digite a sua senha para confirmar"),
    confirm: dialog.getByRole("button", { name: "Excluir" }),
  };
}

describe("DeleteAccount", () => {
  it("deletes the account with the password, then goes to the sign-in", async () => {
    deleteAccountMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(<DeleteAccount blockers={[]} />);

    const { password, confirm } = await openDialog(user);
    expect(confirm.hasAttribute("disabled")).toBe(true);
    await user.type(password, "secret-password");
    await user.click(confirm);

    expect(deleteAccountMock).toHaveBeenCalledWith({
      password: "secret-password",
    });
    expect(await screen.findByText("Conta excluída.")).toBeTruthy();
    expect(replace).toHaveBeenCalledWith("/sign-in");
  });

  it("keeps the dialog open on a wrong password", async () => {
    deleteAccountMock.mockResolvedValue({
      kind: "error",
      code: "invalidPassword",
    });
    const user = renderWithMessages(<DeleteAccount blockers={[]} />);

    const { password, confirm } = await openDialog(user);
    await user.type(password, "wrong");
    await user.click(confirm);

    expect(await screen.findByText("A senha está incorreta.")).toBeTruthy();
    expect(
      screen.getByRole("alertdialog", { name: "Excluir a sua conta?" })
    ).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("lists the blocking Organizations and opens the transfer of each", async () => {
    setActive.mockResolvedValue({ data: null, error: null });
    const user = renderWithMessages(
      <DeleteAccount
        blockers={[
          { id: "org-acme", name: "Acme" },
          { id: "org-globex", name: "Globex" },
        ]}
      />
    );

    const list = screen.getByRole("list", {
      name: "Organizations que impedem a exclusão",
    });
    expect(
      within(list)
        .getAllByRole("link")
        .map((link) => link.textContent)
    ).toEqual([
      "Transferir a propriedade de Acme",
      "Transferir a propriedade de Globex",
    ]);
    expect(
      screen
        .getByRole("button", { name: "Excluir conta" })
        .hasAttribute("disabled")
    ).toBe(true);

    await user.click(
      screen.getByRole("link", { name: "Transferir a propriedade de Globex" })
    );

    expect(setActive).toHaveBeenCalledWith({ organizationId: "org-globex" });
    expect(push).toHaveBeenCalledWith("/settings/organization");
  });
});

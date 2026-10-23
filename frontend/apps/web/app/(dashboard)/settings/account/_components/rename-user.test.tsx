/** @vitest-environment jsdom */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { renameUser } from "@/actions/account";

import {
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { RenameUser } from "./rename-user";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/account", () => ({ renameUser: vi.fn() }));

const renameUserMock = vi.mocked(renameUser);

beforeEach(() => {
  stubBrowserApis();
  renameUserMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function nameField() {
  return screen.getByRole<HTMLInputElement>("textbox", { name: "Seu nome" });
}

describe("RenameUser", () => {
  it("renames the User and refreshes the page", async () => {
    renameUserMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(<RenameUser name="Ana" />);

    expect(nameField().value).toBe("Ana");
    await user.clear(nameField());
    await user.type(nameField(), "  Ana Lima ");
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(renameUserMock).toHaveBeenCalledWith({ name: "Ana Lima" });
    expect(await screen.findByText("Nome atualizado.")).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("asks for a name before calling the server", async () => {
    const user = renderWithMessages(<RenameUser name="Ana" />);

    await user.clear(nameField());
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(nameField().getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText("Campo obrigatório").id).toBe(
      nameField().getAttribute("aria-describedby")
    );
    expect(renameUserMock).not.toHaveBeenCalled();
  });
});

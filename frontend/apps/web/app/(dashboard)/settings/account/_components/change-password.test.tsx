/** @vitest-environment jsdom */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { authClient } from "@/shared/auth/client";

import {
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { ChangePassword } from "./change-password";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/providers", () => ({
  useOrganizations: () => ({
    active: { id: "org-globex", name: "Globex" },
    organizations: [],
  }),
}));
vi.mock("@/shared/auth/client", () => ({
  authClient: {
    changePassword: vi.fn(),
    organization: { setActive: vi.fn() },
  },
}));

const changePasswordMock = vi.mocked(authClient.changePassword);
const setActive = vi.mocked(authClient.organization.setActive);

beforeEach(() => {
  stubBrowserApis();
  changePasswordMock.mockReset();
  setActive.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function field(label: string) {
  return screen.getByLabelText<HTMLInputElement>(label);
}

describe("ChangePassword", () => {
  it("changes the password ending the other sessions, keeps the active Organization and clears the fields", async () => {
    changePasswordMock.mockResolvedValue({ data: null, error: null });
    setActive.mockResolvedValue({ data: null, error: null });
    const user = renderWithMessages(<ChangePassword />);

    await user.type(field("Senha atual"), "old-password");
    await user.type(field("Nova senha"), "new-password");
    await user.click(screen.getByRole("button", { name: "Trocar senha" }));

    expect(changePasswordMock).toHaveBeenCalledWith({
      currentPassword: "old-password",
      newPassword: "new-password",
      revokeOtherSessions: true,
    });
    expect(
      await screen.findByText(
        "Senha trocada. As outras sessões foram encerradas."
      )
    ).toBeTruthy();
    expect(setActive).toHaveBeenCalledWith({ organizationId: "org-globex" });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(field("Senha atual").value).toBe("");
    expect(field("Nova senha").value).toBe("");
  });

  it("explains a wrong current password", async () => {
    changePasswordMock.mockResolvedValue({
      data: null,
      error: {
        code: "INVALID_PASSWORD",
        message: "Invalid password",
        status: 400,
        statusText: "BAD_REQUEST",
      },
    });
    const user = renderWithMessages(<ChangePassword />);

    await user.type(field("Senha atual"), "wrong-password");
    await user.type(field("Nova senha"), "new-password");
    await user.click(screen.getByRole("button", { name: "Trocar senha" }));

    expect(
      await screen.findByText("A senha atual está incorreta.")
    ).toBeTruthy();
    expect(field("Senha atual").value).toBe("wrong-password");
    expect(setActive).not.toHaveBeenCalled();
  });

  it("checks both fields before calling the server", async () => {
    const user = renderWithMessages(<ChangePassword />);

    await user.type(field("Nova senha"), "short");
    await user.click(screen.getByRole("button", { name: "Trocar senha" }));

    expect(field("Senha atual").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText("Campo obrigatório")).toBeTruthy();
    expect(field("Nova senha").getAttribute("aria-invalid")).toBe("true");
    expect(changePasswordMock).not.toHaveBeenCalled();
  });
});

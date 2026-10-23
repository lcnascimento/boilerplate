/** @vitest-environment jsdom */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { renameOrganization } from "@/actions/organization";

import {
  renderWithMessages,
  stubBrowserApis,
} from "../../members/_components/test-utils";

import { RenameOrganization } from "./rename-organization";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/organization", () => ({ renameOrganization: vi.fn() }));

const renameOrganizationMock = vi.mocked(renameOrganization);

beforeEach(() => {
  stubBrowserApis();
  renameOrganizationMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function nameField() {
  return screen.getByRole<HTMLInputElement>("textbox", {
    name: "Nome da Organization",
  });
}

describe("RenameOrganization", () => {
  it("renames the Organization and refreshes the page", async () => {
    renameOrganizationMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(
      <RenameOrganization canRename name="Acme" />
    );

    expect(nameField().value).toBe("Acme");
    await user.clear(nameField());
    await user.type(nameField(), "  Acme Corp ");
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(renameOrganizationMock).toHaveBeenCalledWith({ name: "Acme Corp" });
    expect(await screen.findByText("Organization renomeada.")).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("asks for a name before calling the server", async () => {
    const user = renderWithMessages(
      <RenameOrganization canRename name="Acme" />
    );

    await user.clear(nameField());
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(nameField().getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText("Campo obrigatório").id).toBe(
      nameField().getAttribute("aria-describedby")
    );
    expect(renameOrganizationMock).not.toHaveBeenCalled();
  });

  it("explains a refused rename", async () => {
    renameOrganizationMock.mockResolvedValue({
      kind: "error",
      code: "forbidden",
    });
    const user = renderWithMessages(
      <RenameOrganization canRename name="Acme" />
    );

    await user.type(nameField(), " Corp");
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(
      await screen.findByText("Sua Role não permite renomear a Organization.")
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("shows the name read-only to a Role that cannot rename", () => {
    renderWithMessages(<RenameOrganization canRename={false} name="Acme" />);

    expect(nameField().value).toBe("Acme");
    expect(nameField().readOnly).toBe(true);
    expect(
      screen.getByText("Só o Owner e os Admins podem renomear a Organization.")
        .id
    ).toBe(nameField().getAttribute("aria-describedby"));
    expect(screen.queryByRole("button", { name: "Salvar" })).toBeNull();
  });
});

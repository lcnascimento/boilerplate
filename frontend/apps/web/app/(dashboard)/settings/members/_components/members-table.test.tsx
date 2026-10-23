/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { removeMember, updateMemberRole } from "@/actions/members";
import { memberPolicy, type Member } from "@/shared/auth/members";

import { MembersTable } from "./members-table";
import {
  acme,
  ana,
  bruno,
  carla,
  renderWithMessages,
  rowOf,
  stubBrowserApis,
} from "./test-utils";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/members", () => ({
  removeMember: vi.fn(),
  updateMemberRole: vi.fn(),
}));

const removeMemberMock = vi.mocked(removeMember);
const updateMemberRoleMock = vi.mocked(updateMemberRole);

function renderTable(actor: Member) {
  return renderWithMessages(
    <MembersTable actorId={actor.id} policy={memberPolicy(actor, acme)} />
  );
}

beforeEach(() => {
  stubBrowserApis();
  removeMemberMock.mockReset();
  updateMemberRoleMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("MembersTable", () => {
  it("shows a Viewer every Member with name, email and Role but no actions", () => {
    renderTable(carla);

    const table = screen.getByRole("table", {
      name: "Members da Organization",
    });
    expect(
      within(table)
        .getAllByRole("row")
        .slice(1)
        .map((tr) =>
          within(tr)
            .getAllByRole("cell")
            .map((td) => td.textContent)
        )
    ).toEqual([
      ["Ana Owner", "owner@acme.test", "Owner", ""],
      ["Bruno Admin", "admin@acme.test", "Admin", ""],
      ["Carla Viewer (você)", "viewer@acme.test", "Viewer", ""],
    ]);
    expect(screen.queryByRole("combobox")).toBeNull();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("lets an Admin manage the Viewer but not the Owner", () => {
    renderTable(bruno);

    expect(rowOf("Ana Owner").queryByRole("combobox")).toBeNull();
    expect(rowOf("Ana Owner").queryByRole("button")).toBeNull();
    expect(
      rowOf("Carla Viewer").getByRole("combobox", {
        name: "Role de Carla Viewer",
      })
    ).toBeTruthy();
    expect(
      rowOf("Carla Viewer").getByRole("button", {
        name: "Remover Carla Viewer",
      })
    ).toBeTruthy();
  });

  it("offers an Admin only Admin and Viewer when changing a Role", async () => {
    updateMemberRoleMock.mockResolvedValue({ kind: "ok" });
    const user = renderTable(bruno);

    await user.click(
      screen.getByRole("combobox", { name: "Role de Carla Viewer" })
    );
    expect(
      screen.getAllByRole("option").map((option) => option.textContent)
    ).toEqual(["Admin", "Viewer"]);

    await user.click(screen.getByRole("option", { name: "Admin" }));

    expect(updateMemberRoleMock).toHaveBeenCalledWith({
      memberId: "m-carla",
      role: "admin",
    });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("removes a Member only after confirmation", async () => {
    removeMemberMock.mockResolvedValue({ kind: "ok" });
    const user = renderTable(ana);

    await user.click(
      screen.getByRole("button", { name: "Remover Bruno Admin" })
    );
    const dialog = screen.getByRole("alertdialog", {
      name: "Remover Bruno Admin?",
    });
    expect(removeMemberMock).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("button", { name: "Remover" }));

    expect(removeMemberMock).toHaveBeenCalledWith({ memberId: "m-bruno" });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains a refused Role change", async () => {
    updateMemberRoleMock.mockResolvedValue({
      kind: "error",
      code: "forbidden",
    });
    const user = renderTable(bruno);

    await user.click(
      screen.getByRole("combobox", { name: "Role de Carla Viewer" })
    );
    await user.click(screen.getByRole("option", { name: "Admin" }));

    expect(
      await screen.findByText("Sua Role não permite essa ação.")
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });
});

/** @vitest-environment jsdom */
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { cancelInvitation } from "@/actions/invitations";

import { InvitationsTable } from "./invitations-table";
import { renderWithMessages, rowOf, stubBrowserApis } from "./test-utils";

import type { SentInvitation } from "@/shared/auth/invitations";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/invitations", () => ({ cancelInvitation: vi.fn() }));

const cancelInvitationMock = vi.mocked(cancelInvitation);

const invitations: SentInvitation[] = [
  {
    id: "7a1d0c3e-0000-4000-8000-000000000001",
    email: "elisa@globex.test",
    role: "admin",
    expiresAt: new Date("2026-10-29T15:00:00Z"),
  },
  {
    id: "7a1d0c3e-0000-4000-8000-000000000002",
    email: "fabio@globex.test",
    role: "viewer",
    expiresAt: new Date("2026-10-30T15:00:00Z"),
  },
];

beforeEach(() => {
  stubBrowserApis();
  cancelInvitationMock.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("InvitationsTable", () => {
  it("lists each pending invitation with its Role and expiry", () => {
    renderWithMessages(
      <InvitationsTable canCancel={false} invitations={invitations} />
    );

    expect(
      screen.getByRole("heading", { name: "Convites pendentes" })
    ).toBeTruthy();
    expect(
      screen
        .getAllByRole("row")
        .slice(1)
        .map((row) => row.textContent)
    ).toEqual([
      "elisa@globex.testAdmin29/10/2026",
      "fabio@globex.testViewer30/10/2026",
    ]);
    expect(screen.queryAllByRole("button")).toEqual([]);
  });

  it("cancels an invitation and refreshes", async () => {
    cancelInvitationMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(
      <InvitationsTable canCancel invitations={invitations} />
    );

    await user.click(
      rowOf("fabio@globex.test").getByRole("button", {
        name: "Cancelar o convite de fabio@globex.test",
      })
    );

    expect(cancelInvitationMock).toHaveBeenCalledWith({
      invitationId: "7a1d0c3e-0000-4000-8000-000000000002",
    });
    expect(await screen.findByText("Convite cancelado.")).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains a failed cancel", async () => {
    cancelInvitationMock.mockResolvedValue({
      kind: "error",
      code: "forbidden",
    });
    const user = renderWithMessages(
      <InvitationsTable canCancel invitations={invitations} />
    );

    await user.click(
      rowOf("elisa@globex.test").getByRole("button", {
        name: "Cancelar o convite de elisa@globex.test",
      })
    );

    expect(
      await screen.findByText("Sua Role não permite essa ação.")
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("renders nothing without pending invitations", () => {
    renderWithMessages(<InvitationsTable canCancel invitations={[]} />);

    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("table")).toBeNull();
  });
});

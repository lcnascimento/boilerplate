/** @vitest-environment jsdom */
import { cleanup, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { leaveOrganization } from "@/actions/members";

import { LeaveOrganization } from "./leave-organization";
import { renderWithMessages, stubBrowserApis } from "./test-utils";

const push = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh }) }));
vi.mock("@/actions/members", () => ({ leaveOrganization: vi.fn() }));

const leaveOrganizationMock = vi.mocked(leaveOrganization);

beforeEach(() => {
  stubBrowserApis();
  leaveOrganizationMock.mockReset();
  push.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("LeaveOrganization", () => {
  it("tells the last Owner why they cannot leave", () => {
    renderWithMessages(<LeaveOrganization leave="lastOwner" />);

    const button = screen.getByRole("button", {
      name: "Sair da Organization",
    });
    expect(button.hasAttribute("disabled")).toBe(true);
    expect(
      screen.getByText(
        "Você é o único Owner. Promova outro Member a Owner antes de sair."
      ).id
    ).toBe(button.getAttribute("aria-describedby"));
  });

  it("leaves after confirmation and goes home", async () => {
    leaveOrganizationMock.mockResolvedValue({ kind: "ok" });
    const user = renderWithMessages(<LeaveOrganization leave="allowed" />);

    await user.click(
      screen.getByRole("button", { name: "Sair da Organization" })
    );
    const dialog = screen.getByRole("alertdialog", {
      name: "Sair da Organization?",
    });
    await user.click(within(dialog).getByRole("button", { name: "Sair" }));

    expect(leaveOrganizationMock).toHaveBeenCalledTimes(1);
    expect(push).toHaveBeenCalledWith("/");
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains a refusal from the last-Owner rule", async () => {
    leaveOrganizationMock.mockResolvedValue({
      kind: "error",
      code: "lastOwner",
    });
    const user = renderWithMessages(<LeaveOrganization leave="allowed" />);

    await user.click(
      screen.getByRole("button", { name: "Sair da Organization" })
    );
    await user.click(screen.getByRole("button", { name: "Sair" }));

    expect(
      await screen.findByText("A Organization precisa de pelo menos um Owner.")
    ).toBeTruthy();
    expect(push).not.toHaveBeenCalled();
  });
});

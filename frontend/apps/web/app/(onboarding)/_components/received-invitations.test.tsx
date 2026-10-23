/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { acceptInvitation, rejectInvitation } from "@/actions/invitations";
import { appMessages } from "@/shared/_messages";
import { SIGNED_IN_HOME } from "@/shared/auth/routes";

import { ReceivedInvitations } from "./received-invitations";

import type { ReceivedInvitation } from "@/shared/auth/invitations";

const replace = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, refresh }),
}));
vi.mock("@/actions/invitations", () => ({
  acceptInvitation: vi.fn(),
  rejectInvitation: vi.fn(),
}));

const acceptMock = vi.mocked(acceptInvitation);
const rejectMock = vi.mocked(rejectInvitation);

const GLOBEX_INVITATION = "7a1d0c3e-0000-4000-8000-000000000011";
const invitations: ReceivedInvitation[] = [
  {
    id: GLOBEX_INVITATION,
    organizationName: "Globex",
    role: "admin",
    expiresAt: new Date("2026-10-29T15:00:00Z"),
  },
  {
    id: "7a1d0c3e-0000-4000-8000-000000000012",
    organizationName: "Initech",
    role: "viewer",
    expiresAt: new Date("2026-10-30T15:00:00Z"),
  },
];

function renderInvitations(list: readonly ReceivedInvitation[] = invitations) {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }));
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <ReceivedInvitations invitations={list} />
      <Toaster />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

beforeEach(() => {
  acceptMock.mockReset();
  rejectMock.mockReset();
  replace.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("ReceivedInvitations", () => {
  it("lists each Organization with the Role and the expiry", () => {
    renderInvitations();

    expect(
      screen.getAllByRole("listitem").map((item) => item.textContent)
    ).toEqual([
      "GlobexComo Admin, até 29/10/2026AceitarRecusar",
      "InitechComo Viewer, até 30/10/2026AceitarRecusar",
    ]);
  });

  it("accepts an invitation and goes to the signed-in home", async () => {
    acceptMock.mockResolvedValue({ kind: "ok" });
    const user = renderInvitations();

    await user.click(
      screen.getByRole("button", { name: "Aceitar o convite de Globex" })
    );

    expect(acceptMock).toHaveBeenCalledWith({
      invitationId: GLOBEX_INVITATION,
    });
    expect(replace).toHaveBeenCalledWith(SIGNED_IN_HOME);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains the limit when the User already has 10 Organizations", async () => {
    acceptMock.mockResolvedValue({ kind: "error", code: "organizationLimit" });
    const user = renderInvitations();

    await user.click(
      screen.getByRole("button", { name: "Aceitar o convite de Globex" })
    );

    expect(
      await screen.findByText(
        "Você já participa de 10 Organizations, o máximo por User. Saia de uma delas para aceitar este convite."
      )
    ).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("declines an invitation and refreshes the list", async () => {
    rejectMock.mockResolvedValue({ kind: "ok" });
    const user = renderInvitations();

    await user.click(
      screen.getByRole("button", { name: "Recusar o convite de Initech" })
    );

    expect(rejectMock).toHaveBeenCalledWith({
      invitationId: "7a1d0c3e-0000-4000-8000-000000000012",
    });
    expect(await screen.findByText("Convite recusado.")).toBeTruthy();
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(replace).not.toHaveBeenCalled();
  });

  it("renders nothing without invitations", () => {
    renderInvitations([]);

    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("list")).toBeNull();
  });
});

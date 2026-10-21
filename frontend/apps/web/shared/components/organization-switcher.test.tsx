/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";
import { SidebarProvider } from "@boilerplate/ui/components/sidebar";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { appMessages } from "@/shared/_messages";
import { authClient } from "@/shared/auth/client";

import { OrganizationSwitcher } from "./organization-switcher";

import type { UserOrganizations } from "@/shared/auth/organizations";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/shared/auth/client", () => ({
  authClient: { organization: { setActive: vi.fn() } },
}));

const setActive = vi.mocked(authClient.organization.setActive);

const acme = { id: "org-acme", name: "Acme" };
const globex = { id: "org-globex", name: "Globex" };
const acmeActive: UserOrganizations = {
  active: acme,
  organizations: [acme, globex],
};

function renderSwitcher(organizations: UserOrganizations) {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <SidebarProvider>
        <OrganizationSwitcher organizations={organizations} />
      </SidebarProvider>
      <Toaster />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
  }));
  setActive.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("OrganizationSwitcher", () => {
  it("shows the active Organization and lists every one, the active checked", async () => {
    const user = renderSwitcher(acmeActive);

    const trigger = screen.getByRole("button", {
      name: "Trocar de Organization",
    });
    expect(trigger.textContent).toContain("Acme");
    await user.click(trigger);

    const items = screen.getAllByRole("menuitemradio");
    expect(
      items.map((item) => [item.textContent, item.getAttribute("aria-checked")])
    ).toEqual([
      ["Acme", "true"],
      ["Globex", "false"],
    ]);
  });

  it("makes the chosen Organization active and refreshes the page", async () => {
    setActive.mockResolvedValue({ data: null, error: null });
    const user = renderSwitcher(acmeActive);

    await user.click(
      screen.getByRole("button", { name: "Trocar de Organization" })
    );
    await user.click(screen.getByRole("menuitemradio", { name: "Globex" }));

    expect(setActive).toHaveBeenCalledWith({ organizationId: "org-globex" });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("stays put when the active Organization is chosen again", async () => {
    const user = renderSwitcher(acmeActive);

    await user.click(
      screen.getByRole("button", { name: "Trocar de Organization" })
    );
    await user.click(screen.getByRole("menuitemradio", { name: "Acme" }));

    expect(setActive).not.toHaveBeenCalled();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("says so when the switch fails", async () => {
    setActive.mockResolvedValue({
      data: null,
      error: { status: 403, statusText: "Forbidden" },
    });
    const user = renderSwitcher(acmeActive);

    await user.click(
      screen.getByRole("button", { name: "Trocar de Organization" })
    );
    await user.click(screen.getByRole("menuitemradio", { name: "Globex" }));

    expect(
      await screen.findByText(
        "Não foi possível trocar de Organization. Tente de novo."
      )
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });
});

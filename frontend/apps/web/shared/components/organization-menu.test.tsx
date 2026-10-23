/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";
import { SidebarProvider } from "@boilerplate/ui/components/sidebar";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { OrganizationsProvider } from "@/providers";
import { appMessages } from "@/shared/_messages";
import { authClient } from "@/shared/auth/client";

import { OrganizationMenu } from "./organization-menu";

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

function renderMenu(organizations: UserOrganizations) {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <OrganizationsProvider organizations={organizations}>
        <SidebarProvider>
          <OrganizationMenu />
        </SidebarProvider>
      </OrganizationsProvider>
      <Toaster />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

async function openSwitch(user: ReturnType<typeof userEvent.setup>) {
  await user.click(
    screen.getByRole("button", { name: "Abrir menu da Organization" })
  );
  await user.click(
    screen.getByRole("menuitem", { name: "Trocar Organization" })
  );
}

async function choose(
  user: ReturnType<typeof userEvent.setup>,
  organization: string
) {
  screen.getByRole("menuitemradio", { name: organization }).focus();
  await user.keyboard("{Enter}");
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

describe("OrganizationMenu", () => {
  it("shows the active Organization and links to its settings and Members", async () => {
    const user = renderMenu(acmeActive);

    const trigger = screen.getByRole("button", {
      name: "Abrir menu da Organization",
    });
    expect(trigger.textContent).toContain("Acme");
    await user.click(trigger);

    expect(
      screen
        .getByRole("menuitem", { name: "Configurações" })
        .getAttribute("href")
    ).toBe("/settings/organization");
    expect(
      screen.getByRole("menuitem", { name: "Members" }).getAttribute("href")
    ).toBe("/settings/members");
  });

  it("lists every Organization under the switch, the active checked", async () => {
    const user = renderMenu(acmeActive);

    await openSwitch(user);

    const items = screen.getAllByRole("menuitemradio");
    expect(
      items.map((item) => [item.textContent, item.getAttribute("aria-checked")])
    ).toEqual([
      ["Acme", "true"],
      ["Globex", "false"],
    ]);
  });

  it("ends the switch with a link to create or join an Organization", async () => {
    const user = renderMenu(acmeActive);

    await openSwitch(user);

    const submenu = screen.getAllByRole("menu").at(-1);
    const items = Array.from(
      submenu?.querySelectorAll("[role^=menuitem]") ?? [],
      (item) => item.textContent
    );
    expect(items).toEqual(["Acme", "Globex", "Criar ou entrar"]);
    expect(
      screen
        .getByRole("menuitem", { name: "Criar ou entrar" })
        .getAttribute("href")
    ).toBe("/organizations/new");
  });

  it("makes the chosen Organization active and refreshes the page", async () => {
    setActive.mockResolvedValue({ data: null, error: null });
    const user = renderMenu(acmeActive);

    await openSwitch(user);
    await choose(user, "Globex");

    expect(setActive).toHaveBeenCalledWith({ organizationId: "org-globex" });
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("stays put when the active Organization is chosen again", async () => {
    const user = renderMenu(acmeActive);

    await openSwitch(user);
    await choose(user, "Acme");

    expect(setActive).not.toHaveBeenCalled();
    expect(refresh).not.toHaveBeenCalled();
  });

  it("says so when the switch fails", async () => {
    setActive.mockResolvedValue({
      data: null,
      error: { status: 403, statusText: "Forbidden" },
    });
    const user = renderMenu(acmeActive);

    await openSwitch(user);
    await choose(user, "Globex");

    expect(
      await screen.findByText(
        "Não foi possível trocar de Organization. Tente de novo."
      )
    ).toBeTruthy();
    expect(refresh).not.toHaveBeenCalled();
  });
});

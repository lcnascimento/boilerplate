/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { appMessages } from "@/shared/_messages";
import { authClient } from "@/shared/auth/client";

import { NoOrganization } from "./no-organization";

const replace = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, refresh }),
}));
vi.mock("@/shared/auth/client", () => ({
  authClient: { signOut: vi.fn() },
}));

const signOut = vi.mocked(authClient.signOut);

function renderScreen() {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <NoOrganization />
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
  }));
  signOut.mockReset();
  replace.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("NoOrganization", () => {
  it("explains that the User has no Organization", () => {
    renderScreen();

    expect(
      screen.getByRole("heading", {
        name: "Você não faz parte de nenhuma Organization",
      })
    ).toBeTruthy();
  });

  it("signs out to the sign-in page", async () => {
    signOut.mockResolvedValue({ data: { success: true }, error: null });
    const user = renderScreen();

    await user.click(screen.getByRole("button", { name: "Sair" }));

    expect(signOut).toHaveBeenCalledTimes(1);
    expect(replace).toHaveBeenCalledWith("/sign-in");
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("stays when signing out fails, and says so", async () => {
    signOut.mockResolvedValue({
      data: null,
      error: { status: 500, statusText: "Internal Server Error" },
    });
    const user = renderScreen();

    await user.click(screen.getByRole("button", { name: "Sair" }));

    expect(
      await screen.findByText("Não foi possível sair. Tente de novo.")
    ).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });
});

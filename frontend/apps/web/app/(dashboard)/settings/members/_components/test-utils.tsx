import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { vi } from "vitest";

import { messages } from "@boilerplate/i18n";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { appMessages } from "@/shared/_messages";

import type { Member } from "@/shared/auth/members";
import type { ReactNode } from "react";

export const ana: Member = {
  id: "m-ana",
  name: "Ana Owner",
  email: "owner@acme.test",
  role: "owner",
};
export const bruno: Member = {
  id: "m-bruno",
  name: "Bruno Admin",
  email: "admin@acme.test",
  role: "admin",
};
export const carla: Member = {
  id: "m-carla",
  name: "Carla Viewer",
  email: "viewer@acme.test",
  role: "viewer",
};
export const acme = [ana, bruno, carla];

// jsdom lacks what Radix Select and sonner touch.
export function stubBrowserApis() {
  Element.prototype.hasPointerCapture = () => false;
  Element.prototype.scrollIntoView = () => undefined;
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
  }));
}

export function renderWithMessages(ui: ReactNode) {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      {ui}
      <Toaster />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

export function rowOf(name: string) {
  const match = screen
    .getAllByRole("row")
    .find((tr) => tr.textContent.includes(name));
  if (match === undefined) throw new Error(`no row for ${name}`);
  return within(match);
}

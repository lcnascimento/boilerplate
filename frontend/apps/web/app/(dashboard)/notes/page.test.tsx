/** @vitest-environment jsdom */
import { act, cleanup, render, screen, within } from "@testing-library/react";
import { createTranslator, NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";

import { getActiveRole } from "@/actions/members";
import { listNotes } from "@/actions/notes";
import { appMessages } from "@/shared/_messages";

import NotesPage from "./page";

const allMessages = { ...messages["pt-BR"], ...appMessages("pt-BR") };

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh: vi.fn() }) }));
vi.mock("next-intl/server", () => ({
  getTranslations: (namespace: "notes") =>
    Promise.resolve(
      createTranslator({ locale: "pt-BR", messages: allMessages, namespace })
    ),
}));
vi.mock("@/actions/notes", () => ({ listNotes: vi.fn() }));
vi.mock("@/actions/members", () => ({ getActiveRole: vi.fn() }));

const listNotesMock = vi.mocked(listNotes);
const getActiveRoleMock = vi.mocked(getActiveRole);

// Awaiting the page yields sync components; the awaited act lets Suspense settle.
async function renderPage() {
  await act(async () => {
    const page = await NotesPage();
    render(
      <NextIntlClientProvider
        locale="pt-BR"
        messages={allMessages}
        timeZone="America/Sao_Paulo"
      >
        {page}
      </NextIntlClientProvider>
    );
  });
}

beforeEach(() => {
  listNotesMock.mockReset();
  getActiveRoleMock.mockReset();
  getActiveRoleMock.mockResolvedValue("owner");
});

afterEach(cleanup);

describe("NotesPage", () => {
  it.each(["owner", "admin"] as const)(
    "offers an %s the New note button",
    async (role) => {
      getActiveRoleMock.mockResolvedValue(role);
      listNotesMock.mockResolvedValue({ kind: "ok", data: [] });

      await renderPage();

      expect(screen.getByRole("button", { name: "Nova nota" })).toBeTruthy();
      expect(
        await screen.findByText("Crie a primeira pelo botão Nova nota.")
      ).toBeTruthy();
    }
  );

  it.each(["viewer", null] as const)(
    "hides the New note button from a %s",
    async (role) => {
      getActiveRoleMock.mockResolvedValue(role);
      listNotesMock.mockResolvedValue({ kind: "ok", data: [] });

      await renderPage();

      expect(
        await screen.findByText(
          "As notas que Owners e Admins criarem aparecem aqui."
        )
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Nova nota" })).toBeNull();
    }
  );

  it("shows the empty state when there are no notes", async () => {
    listNotesMock.mockResolvedValue({ kind: "ok", data: [] });

    await renderPage();

    expect(
      await screen.findByRole("heading", { name: "Nenhuma nota ainda" })
    ).toBeTruthy();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("shows the error state with a retry when the service is unavailable", async () => {
    listNotesMock.mockResolvedValue({
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    });

    await renderPage();

    const alert = await screen.findByRole("alert");
    expect(
      within(alert).getByRole("heading", {
        name: "Serviço de notas indisponível",
      })
    ).toBeTruthy();
    expect(
      within(alert).getByRole("button", { name: "Tentar novamente" })
    ).toBeTruthy();
    expect(
      screen.queryByRole("heading", { name: "Nenhuma nota ainda" })
    ).toBeNull();
  });
});

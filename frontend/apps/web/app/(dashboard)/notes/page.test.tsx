/** @vitest-environment jsdom */
import { act, cleanup, render, screen, within } from "@testing-library/react";
import { createTranslator, NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";

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

const listNotesMock = vi.mocked(listNotes);

// The page is async only for its translations: awaiting it yields a tree of
// sync components, and Notes resolves listNotes under the page's Suspense,
// which the awaited act lets settle.
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
});

afterEach(cleanup);

describe("NotesPage", () => {
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

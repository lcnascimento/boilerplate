/** @vitest-environment jsdom */
import { act, cleanup, render, screen } from "@testing-library/react";
import {
  createFormatter,
  createTranslator,
  NextIntlClientProvider,
} from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { noteIdSchema } from "@boilerplate/api/notes/schemas";
import { messages } from "@boilerplate/i18n";

import { getActiveRole } from "@/actions/members";
import { getNote } from "@/actions/notes";
import { appMessages } from "@/shared/_messages";

import NotePage from "./page";

const allMessages = { ...messages["pt-BR"], ...appMessages("pt-BR") };
const ID = "note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }),
}));
vi.mock("next-intl/server", () => ({
  getTranslations: (namespace: "notes") =>
    Promise.resolve(
      createTranslator({ locale: "pt-BR", messages: allMessages, namespace })
    ),
  getFormatter: () =>
    Promise.resolve(
      createFormatter({ locale: "pt-BR", timeZone: "America/Sao_Paulo" })
    ),
}));
vi.mock("@/actions/notes", () => ({ getNote: vi.fn() }));
vi.mock("@/actions/members", () => ({ getActiveRole: vi.fn() }));

const getNoteMock = vi.mocked(getNote);
const getActiveRoleMock = vi.mocked(getActiveRole);

async function renderPage() {
  await act(async () => {
    const page = await NotePage({ params: Promise.resolve({ id: ID }) });
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
  getActiveRoleMock.mockReset();
  getNoteMock.mockReset();
  getNoteMock.mockResolvedValue({
    kind: "ok",
    data: {
      id: noteIdSchema.parse(ID),
      title: "Compras",
      body: "leite",
      createdAt: new Date("2026-10-10T12:30:00Z"),
    },
  });
});

afterEach(cleanup);

describe("NotePage", () => {
  it("offers an Owner the delete action", async () => {
    getActiveRoleMock.mockResolvedValue("owner");

    await renderPage();

    expect(screen.getByRole("heading", { name: "Compras" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Excluir" })).toBeTruthy();
  });

  it.each(["admin", "viewer", null] as const)(
    "hides the delete action from a %s",
    async (role) => {
      getActiveRoleMock.mockResolvedValue(role);

      await renderPage();

      expect(screen.getByRole("heading", { name: "Compras" })).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Excluir" })).toBeNull();
    }
  );
});

/** @vitest-environment jsdom */
import { cleanup, render, screen, within } from "@testing-library/react";
import { createTranslator, NextIntlClientProvider } from "next-intl";
import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
} from "react";
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

/**
 * Renders the async server components in a tree ahead of time, as the server
 * would, since the DOM renderer only takes sync components.
 */
async function resolveServerTree(node: ReactNode): Promise<ReactNode> {
  if (Array.isArray(node)) return Promise.all(node.map(resolveServerTree));
  if (!isValidElement<{ children?: ReactNode }>(node)) return node;

  const { type, props } = node;
  if (typeof type === "function" && type.constructor.name === "AsyncFunction") {
    const render = type as (props: unknown) => Promise<ReactNode>;
    return resolveServerTree(await render(props));
  }
  if (props.children === undefined) return node;
  return cloneElement(node as ReactElement<{ children?: ReactNode }>, {
    children: await resolveServerTree(props.children),
  });
}

async function renderPage() {
  const page = await resolveServerTree(<NotesPage />);
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={allMessages}
      timeZone="America/Sao_Paulo"
    >
      {page}
    </NextIntlClientProvider>
  );
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
      screen.getByRole("heading", { name: "Nenhuma nota ainda" })
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

    const alert = screen.getByRole("alert");
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

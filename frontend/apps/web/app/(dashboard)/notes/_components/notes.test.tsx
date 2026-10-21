/** @vitest-environment jsdom */
import { act, cleanup, render, screen, within } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import { Suspense } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";

import { appMessages } from "@/shared/_messages";

import { Notes } from "./notes";

import type { Result } from "@boilerplate/api/core";
import type { Note } from "@boilerplate/api/notes/schemas";

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh: vi.fn() }) }));

async function renderNotes(canCreate: boolean, result: Result<Note[]>) {
  const notes = Promise.resolve(result);
  await act(async () => {
    render(
      <NextIntlClientProvider
        locale="pt-BR"
        messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
        timeZone="America/Sao_Paulo"
      >
        <Suspense>
          <Notes canCreate={canCreate} notes={notes} />
        </Suspense>
      </NextIntlClientProvider>
    );
    await notes;
  });
}

afterEach(cleanup);

describe("Notes", () => {
  it("points who can create to the New note button when there are no notes", async () => {
    await renderNotes(true, { kind: "ok", data: [] });

    expect(
      await screen.findByRole("heading", { name: "Nenhuma nota ainda" })
    ).toBeTruthy();
    expect(
      screen.getByText("Crie a primeira pelo botão Nova nota.")
    ).toBeTruthy();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("tells who cannot create that Owners and Admins add the notes", async () => {
    await renderNotes(false, { kind: "ok", data: [] });

    expect(
      await screen.findByText(
        "As notas que Owners e Admins criarem aparecem aqui."
      )
    ).toBeTruthy();
  });

  it("shows the error state with a retry when the service is unavailable", async () => {
    await renderNotes(true, {
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    });

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

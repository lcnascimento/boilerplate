/** @vitest-environment jsdom */
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { noteIdSchema } from "@boilerplate/api/notes/schemas";
import { messages } from "@boilerplate/i18n";
import { Toaster } from "@boilerplate/ui/components/sonner";

import { deleteNote } from "@/actions/notes";
import { appMessages } from "@/shared/_messages";

import { DeleteNoteDialog } from "./delete-note-dialog";

const push = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh }) }));
vi.mock("@/actions/notes", () => ({ deleteNote: vi.fn() }));

const deleteNoteMock = vi.mocked(deleteNote);
const ID = noteIdSchema.parse("note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f");

function renderDialog() {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <DeleteNoteDialog id={ID} title="Compras" />
      <Toaster />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

async function openConfirmation(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "Excluir" }));
  return screen.getByRole("alertdialog", { name: "Excluir esta nota?" });
}

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  }));
  deleteNoteMock.mockReset();
  push.mockReset();
  refresh.mockReset();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("DeleteNoteDialog", () => {
  it("names the note in the confirmation", async () => {
    const user = renderDialog();

    const confirmation = await openConfirmation(user);

    expect(confirmation.textContent).toContain(
      "A nota “Compras” será excluída. Essa ação não pode ser desfeita."
    );
  });

  it("deletes nothing when cancelled", async () => {
    const user = renderDialog();
    await openConfirmation(user);

    await user.click(screen.getByRole("button", { name: "Cancelar" }));

    expect(screen.queryByRole("alertdialog")).toBeNull();
    expect(deleteNoteMock).not.toHaveBeenCalled();
  });

  it("deletes the note on confirm and returns to the list", async () => {
    deleteNoteMock.mockResolvedValue({ kind: "ok", data: undefined });
    const user = renderDialog();
    const confirmation = await openConfirmation(user);

    await user.click(
      within(confirmation).getByRole("button", { name: "Excluir" })
    );

    expect(deleteNoteMock).toHaveBeenCalledWith(ID);
    expect(push).toHaveBeenCalledWith("/notes");
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("says the Role forbids it when the API answers 403", async () => {
    deleteNoteMock.mockResolvedValue({
      kind: "error",
      error: {
        code: "ERR_UNAUTHORIZED",
        message: "unauthorized request",
        retryable: false,
        status: 403,
      },
    });
    const user = renderDialog();
    const confirmation = await openConfirmation(user);

    await user.click(
      within(confirmation).getByRole("button", { name: "Excluir" })
    );

    expect(
      await screen.findByText("Sua Role não permite excluir notas.")
    ).toBeTruthy();
    expect(push).not.toHaveBeenCalled();
  });

  it("stays on the note when the service fails", async () => {
    deleteNoteMock.mockResolvedValue({
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    });
    const user = renderDialog();
    const confirmation = await openConfirmation(user);

    await user.click(
      within(confirmation).getByRole("button", { name: "Excluir" })
    );

    expect(deleteNoteMock).toHaveBeenCalledWith(ID);
    expect(push).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeTruthy();
  });
});

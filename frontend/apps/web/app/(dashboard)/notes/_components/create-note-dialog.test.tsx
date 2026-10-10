/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { noteIdSchema } from "@boilerplate/api/notes/schemas";
import { messages } from "@boilerplate/i18n";

import { createNote } from "@/actions/notes";
import { appMessages } from "@/shared/_messages";

import { CreateNoteDialog } from "./create-note-dialog";

const refresh = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));
vi.mock("@/actions/notes", () => ({ createNote: vi.fn() }));

const createNoteMock = vi.mocked(createNote);

function renderDialog() {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <CreateNoteDialog />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

async function openDialog(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "Nova nota" }));
  return screen.getByRole("dialog", { name: "Nova nota" });
}

beforeEach(() => {
  createNoteMock.mockReset();
  refresh.mockReset();
});

afterEach(cleanup);

describe("CreateNoteDialog", () => {
  it("requires a title before sending anything", async () => {
    const user = renderDialog();
    await openDialog(user);

    await user.click(screen.getByRole("button", { name: "Criar" }));

    expect(screen.getByLabelText("Título").getAttribute("aria-invalid")).toBe(
      "true"
    );
    expect(screen.getByText("Campo obrigatório")).toBeTruthy();
    expect(createNoteMock).not.toHaveBeenCalled();
  });

  it("limits the title to 120 and the body to 10,000 characters", async () => {
    const user = renderDialog();
    await openDialog(user);

    await user.click(screen.getByLabelText("Título"));
    await user.paste("a".repeat(121));
    await user.click(screen.getByLabelText("Conteúdo"));
    await user.paste("b".repeat(10_001));
    await user.click(screen.getByRole("button", { name: "Criar" }));

    expect(screen.getByText("Máximo de 120 caracteres")).toBeTruthy();
    expect(screen.getByText("Máximo de 10000 caracteres")).toBeTruthy();
    expect(createNoteMock).not.toHaveBeenCalled();
  });

  it("sends the parsed note, closes and refreshes the list", async () => {
    createNoteMock.mockResolvedValue({
      kind: "ok",
      data: {
        id: noteIdSchema.parse("note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f"),
        title: "Compras",
        body: "leite",
        createdAt: new Date("2026-10-10T12:30:00Z"),
      },
    });
    const user = renderDialog();
    await openDialog(user);

    await user.type(screen.getByLabelText("Título"), "  Compras ");
    await user.type(screen.getByLabelText("Conteúdo"), "leite");
    await user.click(screen.getByRole("button", { name: "Criar" }));

    expect(createNoteMock).toHaveBeenCalledWith({
      title: "Compras",
      body: "leite",
    });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("omits an empty body", async () => {
    createNoteMock.mockResolvedValue({
      kind: "error",
      error: { code: "UNKNOWN", message: "x", retryable: false, status: 500 },
    });
    const user = renderDialog();
    await openDialog(user);

    await user.type(screen.getByLabelText("Título"), "Compras");
    await user.click(screen.getByRole("button", { name: "Criar" }));

    expect(createNoteMock).toHaveBeenCalledWith({ title: "Compras" });
  });

  it("stays open and says so when the service is unavailable", async () => {
    createNoteMock.mockResolvedValue({
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    });
    const user = renderDialog();
    const dialog = await openDialog(user);

    await user.type(screen.getByLabelText("Título"), "Compras");
    await user.click(screen.getByRole("button", { name: "Criar" }));

    expect(screen.getByRole("alert").textContent).toBe(
      "Não conseguimos falar com o serviço de notas. Tente de novo em instantes."
    );
    expect(dialog.isConnected).toBe(true);
    expect(refresh).not.toHaveBeenCalled();
  });
});

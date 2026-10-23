/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";

import { createOrganization } from "@/actions/organizations";
import { appMessages } from "@/shared/_messages";
import { SIGNED_IN_HOME } from "@/shared/auth/routes";

import { CreateOrganizationForm } from "./create-organization-form";

const replace = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, refresh }),
}));
vi.mock("@/actions/organizations", () => ({ createOrganization: vi.fn() }));

const createOrganizationMock = vi.mocked(createOrganization);

function renderForm() {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <CreateOrganizationForm />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

async function submit(user: ReturnType<typeof userEvent.setup>, name: string) {
  if (name !== "") {
    await user.type(screen.getByLabelText("Nome da Organization"), name);
  }
  await user.click(screen.getByRole("button", { name: "Criar Organization" }));
}

beforeEach(() => {
  createOrganizationMock.mockReset();
  replace.mockReset();
  refresh.mockReset();
});

afterEach(cleanup);

describe("CreateOrganizationForm", () => {
  it("requires a name before creating anything", async () => {
    const user = renderForm();

    await submit(user, "   ");

    const field = screen.getByLabelText("Nome da Organization");
    expect(field.getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText("Campo obrigatório")).toBeTruthy();
    expect(createOrganizationMock).not.toHaveBeenCalled();
  });

  it("refuses a name longer than 100 characters", async () => {
    const user = renderForm();

    await submit(user, "a".repeat(101));

    expect(screen.getByText("Máximo de 100 caracteres")).toBeTruthy();
    expect(createOrganizationMock).not.toHaveBeenCalled();
  });

  it("creates the Organization with the trimmed name and goes to the signed-in home", async () => {
    createOrganizationMock.mockResolvedValue({ kind: "ok" });
    const user = renderForm();

    await submit(user, "  Initech  ");

    expect(createOrganizationMock).toHaveBeenCalledWith({ name: "Initech" });
    expect(replace).toHaveBeenCalledWith(SIGNED_IN_HOME);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("explains the limit when the User already has 10 Organizations", async () => {
    createOrganizationMock.mockResolvedValue({
      kind: "error",
      code: "organizationLimit",
    });
    const user = renderForm();

    await submit(user, "Initech");

    expect(screen.getByRole("alert").textContent).toBe(
      "Você já participa de 10 Organizations, o máximo por User. Saia de uma delas para criar outra."
    );
    expect(replace).not.toHaveBeenCalled();
  });

  it("says so when creating fails for another reason", async () => {
    createOrganizationMock.mockResolvedValue({
      kind: "error",
      code: "unexpected",
    });
    const user = renderForm();

    await submit(user, "Initech");

    expect(screen.getByRole("alert").textContent).toBe(
      "Não foi possível criar a Organization. Tente de novo."
    );
    expect(replace).not.toHaveBeenCalled();
  });
});

/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { messages } from "@boilerplate/i18n";

import { appMessages } from "@/shared/_messages";
import { authClient } from "@/shared/auth/client";

import { SignUpForm } from "./sign-up-form";

const replace = vi.fn();
const refresh = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, refresh }),
}));
vi.mock("@/shared/auth/client", () => ({
  authClient: { signUp: { email: vi.fn() } },
}));

const signUpMock = vi.mocked(authClient.signUp.email);

type SignUpResult = Awaited<ReturnType<typeof authClient.signUp.email>>;

function failure(status: number, code: string): SignUpResult {
  return {
    data: null,
    error: { status, statusText: "", code, message: code },
  } as SignUpResult;
}

function renderForm() {
  render(
    <NextIntlClientProvider
      locale="pt-BR"
      messages={{ ...messages["pt-BR"], ...appMessages("pt-BR") }}
      timeZone="America/Sao_Paulo"
    >
      <SignUpForm />
    </NextIntlClientProvider>
  );
  return userEvent.setup();
}

async function fillForm(
  user: ReturnType<typeof userEvent.setup>,
  { password = "s3cret-pass" }: { password?: string } = {}
) {
  await user.type(screen.getByLabelText("Nome"), " Ada Lovelace ");
  await user.type(screen.getByLabelText("Email"), "ada@example.test");
  await user.type(screen.getByLabelText("Senha"), password);
  await user.click(screen.getByRole("button", { name: "Criar conta" }));
}

function fieldError(label: string): string | null | undefined {
  const field = screen.getByLabelText(label);
  if (field.getAttribute("aria-invalid") !== "true") return undefined;
  const describedBy = field.getAttribute("aria-describedby");
  return describedBy === null
    ? null
    : document.getElementById(describedBy)?.textContent;
}

beforeEach(() => {
  signUpMock.mockReset();
  replace.mockReset();
  refresh.mockReset();
});

afterEach(cleanup);

describe("SignUpForm", () => {
  it("requires every field before sending anything", async () => {
    const user = renderForm();

    await user.click(screen.getByRole("button", { name: "Criar conta" }));

    for (const label of ["Nome", "Email", "Senha"]) {
      expect(fieldError(label)).toBe("Campo obrigatório");
    }
    expect(signUpMock).not.toHaveBeenCalled();
  });

  it("flags a password shorter than 8 characters on its field", async () => {
    const user = renderForm();

    await fillForm(user, { password: "short" });

    expect(fieldError("Senha")).toBe(
      "A senha precisa ter entre 8 e 128 caracteres."
    );
    expect(signUpMock).not.toHaveBeenCalled();
  });

  it("flags a password the server finds too short on its field", async () => {
    signUpMock.mockResolvedValue(failure(400, "PASSWORD_TOO_SHORT"));
    const user = renderForm();

    await fillForm(user);

    expect(fieldError("Senha")).toBe(
      "A senha precisa ter entre 8 e 128 caracteres."
    );
    expect(replace).not.toHaveBeenCalled();
  });

  it("flags an email already in use on its field", async () => {
    signUpMock.mockResolvedValue(
      failure(422, "USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL")
    );
    const user = renderForm();

    await fillForm(user);

    expect(fieldError("Email")).toBe(
      "Este email já está em uso. Entre ou use outro email."
    );
    expect(replace).not.toHaveBeenCalled();
  });

  it("says so in the form when the sign-up fails for another reason", async () => {
    signUpMock.mockResolvedValue(failure(500, "FAILED_TO_CREATE_USER"));
    const user = renderForm();

    await fillForm(user);

    expect(screen.getByRole("alert").textContent).toBe(
      "Não foi possível criar a conta. Tente de novo."
    );
    expect(replace).not.toHaveBeenCalled();
  });

  it("asks only for name, email and password", () => {
    renderForm();

    expect(screen.getAllByRole("textbox").map((field) => field.id)).toEqual([
      "sign-up-name",
      "sign-up-email",
    ]);
    expect(screen.getByLabelText("Senha")).toBeTruthy();
  });

  it("signs up with the trimmed fields and goes to create or join an Organization", async () => {
    signUpMock.mockResolvedValue({ data: {}, error: null } as SignUpResult);
    const user = renderForm();

    await fillForm(user);

    expect(signUpMock).toHaveBeenCalledWith({
      name: "Ada Lovelace",
      email: "ada@example.test",
      password: "s3cret-pass",
    });
    expect(replace).toHaveBeenCalledWith("/organizations/new");
    expect(refresh).toHaveBeenCalledTimes(1);
  });
});

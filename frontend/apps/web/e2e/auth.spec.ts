import { expect, test, type Page } from "@playwright/test";

import { ACME_OWNER, SEED_PASSWORD } from "../scripts/iam-seed/users";
import { SIGNED_IN_HOME } from "../shared/auth/routes";

import { signedOut } from "./auth-state";

const protectedPages = [
  "/settings/preferences",
  // example:begin
  "/notes",
  // example:end
];

async function signIn(page: Page, email: string, password: string) {
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Senha").fill(password);
  await page.getByRole("button", { name: "Entrar" }).click();
}

test.describe("signed out", () => {
  test.use({ storageState: signedOut });

  test("wrong credentials show the generic message", async ({ page }) => {
    await page.goto("/sign-in");

    await signIn(page, ACME_OWNER.email, "not-the-password");

    await expect(page.locator("form").getByRole("alert")).toHaveText(
      "Email ou senha incorretos."
    );
    await expect(page).toHaveURL(/\/sign-in$/);
  });

  test("an invalid email is flagged on its field", async ({ page }) => {
    await page.goto("/sign-in");

    await signIn(page, "not-an-email", SEED_PASSWORD);

    await expect(page.getByText("Informe um email válido.")).toBeVisible();
    await expect(page.getByLabel("Email")).toHaveAttribute(
      "aria-invalid",
      "true"
    );
  });

  for (const path of protectedPages) {
    test(`opening ${path} leads to the sign-in and back after signing in`, async ({
      page,
    }) => {
      await page.goto(path);

      await expect(page).toHaveURL(
        `/sign-in?callbackUrl=${encodeURIComponent(path)}`
      );

      await signIn(page, ACME_OWNER.email, SEED_PASSWORD);

      await expect(page).toHaveURL(path);
    });
  }

  test("signing out ends the session and returns to the sign-in", async ({
    page,
  }) => {
    await page.goto("/sign-in");
    await signIn(page, ACME_OWNER.email, SEED_PASSWORD);
    await expect(page).toHaveURL(SIGNED_IN_HOME);

    await page.getByRole("button", { name: "Abrir menu do usuário" }).click();
    await expect(page.getByRole("menu")).toContainText(ACME_OWNER.email);
    await page.getByRole("menuitem", { name: "Sair" }).click();

    await expect(page).toHaveURL("/sign-in");
    await page.goto("/settings/preferences");
    await expect(page).toHaveURL(/\/sign-in\?callbackUrl=/);
  });

  test("the session cookie is HttpOnly, Secure and SameSite=Strict", async ({
    request,
  }) => {
    const response = await request.post("/api/auth/sign-in/email", {
      data: { email: ACME_OWNER.email, password: SEED_PASSWORD },
    });
    expect(response.status()).toBe(200);
    expect(response.headers()["set-auth-jwt"]).toBeUndefined();

    const sessionCookie = response
      .headersArray()
      .filter(({ name }) => name.toLowerCase() === "set-cookie")
      .map(({ value }) => value)
      .find((value) => value.includes("session_token="));
    expect(sessionCookie).toBeDefined();
    const attributes = (sessionCookie ?? "")
      .split(";")
      .map((part) => part.trim().toLowerCase());
    expect(attributes).toEqual(
      expect.arrayContaining(["httponly", "secure", "samesite=strict"])
    );

    const session = await request.get("/api/auth/get-session");
    expect(session.status()).toBe(200);
    expect(await session.json()).toMatchObject({
      user: { email: ACME_OWNER.email },
    });
    expect(session.headers()["set-auth-jwt"]).toBeUndefined();
  });

  test("the token route does not exist", async ({ request }) => {
    const response = await request.get("/api/auth/token");

    expect(response.status()).toBe(404);
  });
});

test("a signed-in user opening the sign-in goes to the home", async ({
  page,
}) => {
  await page.goto("/sign-in");

  await expect(page).toHaveURL(SIGNED_IN_HOME);
});

test("the user menu shows the signed-in user", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Abrir menu do usuário" }).click();

  const menu = page.getByRole("menu");
  await expect(menu).toContainText(ACME_OWNER.name);
  await expect(menu).toContainText(ACME_OWNER.email);
});

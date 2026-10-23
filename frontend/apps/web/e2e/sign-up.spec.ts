import { randomUUID } from "node:crypto";

import { expect, test, type Page } from "@playwright/test";

import { ACME_OWNER } from "../scripts/iam-seed/users";
import { SIGNED_IN_HOME } from "../shared/auth/routes";

import { signedOut } from "./auth-state";
import { organizationMenu } from "./organization-menu";

interface Visitor {
  name: string;
  email: string;
  password: string;
}

function newVisitor(): Visitor {
  const id = randomUUID().slice(0, 8);
  return {
    name: `Ada ${id}`,
    email: `e2e-sign-up-${id}@example.test`,
    password: "e2e-password",
  };
}

async function fillSignUp(page: Page, visitor: Visitor) {
  await page.getByLabel("Nome", { exact: true }).fill(visitor.name);
  await page.getByLabel("Email").fill(visitor.email);
  await page.getByLabel("Senha").fill(visitor.password);
  await page.getByRole("button", { name: "Criar conta" }).click();
}

test.use({ storageState: signedOut });

test("a visitor signs up without an Organization, creates one in Criar ou entrar and is its Owner", async ({
  page,
}) => {
  const visitor = newVisitor();
  const organizationName = `Analytical ${randomUUID().slice(0, 8)}`;
  await page.goto("/sign-in");
  await page.getByRole("link", { name: "Cadastre-se" }).click();
  await expect(page).toHaveURL("/sign-up");
  await expect(page.getByLabel("Nome da Organization")).toHaveCount(0);

  await fillSignUp(page, visitor);

  await expect(page).toHaveURL("/organizations/new");
  await expect(
    page.getByRole("heading", { name: "Criar ou entrar" })
  ).toBeVisible();
  const before = await page.request.get("/api/auth/organization/list");
  expect(await before.json()).toEqual([]);

  await page.getByLabel("Nome da Organization").fill(organizationName);
  await page.getByRole("button", { name: "Criar Organization" }).click();

  await expect(page).toHaveURL(SIGNED_IN_HOME);
  await expect(organizationMenu(page)).toContainText(organizationName);
  const active = await page.request.get(
    "/api/auth/organization/get-full-organization"
  );
  expect(active.status()).toBe(200);
  expect(await active.json()).toMatchObject({
    name: organizationName,
    slug: expect.stringMatching(/^analytical-[a-z0-9]+-[a-z0-9]{6}$/),
    members: [{ role: "owner", user: { email: visitor.email } }],
  });
});

test("a User without an Organization is sent to Criar ou entrar from any signed-in page", async ({
  page,
}) => {
  const response = await page.request.post("/api/auth/sign-up/email", {
    data: newVisitor(),
  });
  expect(response.status()).toBe(200);

  await page.goto("/settings/preferences");

  await expect(page).toHaveURL("/organizations/new");
  await expect(organizationMenu(page)).toHaveCount(0);
  await expect(page.getByRole("link", { name: /^Voltar para/ })).toHaveCount(0);
});

test("an email already in use is flagged on its field", async ({ page }) => {
  await page.goto("/sign-up");

  await fillSignUp(page, { ...newVisitor(), email: ACME_OWNER.email });

  await expect(page.getByLabel("Email")).toHaveAttribute(
    "aria-invalid",
    "true"
  );
  await expect(
    page.getByText("Este email já está em uso. Entre ou use outro email.")
  ).toBeVisible();
  await expect(page).toHaveURL("/sign-up");
});

test("a short password is flagged on its field", async ({ page }) => {
  await page.goto("/sign-up");

  await fillSignUp(page, { ...newVisitor(), password: "short" });

  await expect(page.getByLabel("Senha")).toHaveAttribute(
    "aria-invalid",
    "true"
  );
  await expect(
    page.getByText("A senha precisa ter entre 8 e 128 caracteres.")
  ).toBeVisible();
});

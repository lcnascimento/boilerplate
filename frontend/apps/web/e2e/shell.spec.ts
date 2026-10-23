import { expect, test } from "@playwright/test";

import { ACME_OWNER } from "../scripts/iam-seed/users";

import { organizationMenu } from "./organization-menu";

test("home loads and switching the language changes the texts", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { level: 1, name: "Bem-vindo ao Boilerplate" })
  ).toBeVisible();
  await expect(page.getByText(ACME_OWNER.name, { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Idioma" }).click();
  await page.getByRole("menuitemradio", { name: "English" }).click();

  await expect(
    page.getByRole("heading", { level: 1, name: "Welcome to Boilerplate" })
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Home" }).and(page.locator("a[href='/']"))
  ).toHaveCount(1);

  await page.reload();
  await expect(
    page.getByRole("heading", { level: 1, name: "Welcome to Boilerplate" })
  ).toBeVisible();
});

test("a collapsed sidebar stays collapsed after a reload", async ({ page }) => {
  await page.goto("/");
  const sidebar = page.locator('[data-slot="sidebar"][data-state]');
  await expect(sidebar).toHaveAttribute("data-state", "expanded");

  await page
    .getByRole("button", { name: "Recolher ou expandir a barra lateral" })
    .click();
  await expect(sidebar).toHaveAttribute("data-state", "collapsed");

  await page.reload();
  await expect(sidebar).toHaveAttribute("data-state", "collapsed");
});

test("an unknown route shows the not-found inside the shell", async ({
  page,
}) => {
  const response = await page.goto("/rota-que-nao-existe");

  expect(response?.status()).toBe(404);

  await expect(
    page.getByRole("heading", { name: "Página não encontrada." })
  ).toBeVisible();
  await expect(organizationMenu(page)).toBeVisible();
});

test("the main menu holds only the product content", async ({ page }) => {
  await page.goto("/");

  await expect(
    page.locator('[data-slot="sidebar-content"]').getByRole("link")
  ).toHaveText([
    "Início",
    // example:begin
    "Notas",
    // example:end
  ]);
});

test("the Organization menu shows the active Organization and opens its Members", async ({
  page,
}) => {
  await page.goto("/");
  await expect(organizationMenu(page)).toContainText("Acme");

  await organizationMenu(page).click();
  await page.getByRole("menuitem", { name: "Members" }).click();

  await expect(page).toHaveURL("/settings/members");
  await expect(
    page.getByRole("heading", { level: 1, name: "Members" })
  ).toBeVisible();
  const settings = page.getByRole("navigation", { name: "Configurações" });
  await expect(settings.getByRole("list", { name: "Organization" })).toHaveText(
    "Members"
  );
  await expect(settings.getByRole("list", { name: "Conta" })).toHaveText(
    "Preferências"
  );
  await expect(settings.getByRole("link", { name: "Members" })).toHaveAttribute(
    "aria-current",
    "page"
  );
});

test("the User menu opens the Preferences under the settings", async ({
  page,
}) => {
  await page.goto("/");

  await page.getByRole("button", { name: "Abrir menu do usuário" }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["Preferências", "Sair"]);
  await page.getByRole("menuitem", { name: "Preferências" }).click();

  await expect(page).toHaveURL("/settings/preferences");
  await expect(
    page.getByRole("heading", { level: 1, name: "Preferências" })
  ).toBeVisible();
  await expect(
    page
      .getByRole("navigation", { name: "Configurações" })
      .getByRole("link", { name: "Preferências" })
  ).toHaveAttribute("aria-current", "page");
});

test("the old Preferences address no longer exists", async ({ page }) => {
  const response = await page.goto("/profile/preferences");

  expect(response?.status()).toBe(404);
});

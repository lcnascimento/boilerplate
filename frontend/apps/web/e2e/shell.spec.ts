import { expect, test } from "@playwright/test";

import { ACME_OWNER } from "../scripts/iam-seed/users";

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
  await expect(page.getByRole("link", { name: "Boilerplate" })).toBeVisible();
});

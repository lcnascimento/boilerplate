import { randomUUID } from "node:crypto";

import { expect, test, type Page } from "@playwright/test";

// Against the real notes service. Each test creates its own note, with a
// unique title, so tests running in parallel never see each other's data.

function uniqueTitle(prefix: string) {
  return `${prefix} ${randomUUID().slice(0, 8)}`;
}

async function createNote(page: Page, title: string, body: string) {
  await page.goto("/notes");
  await page.getByRole("button", { name: "Nova nota" }).click();

  const dialog = page.getByRole("dialog", { name: "Nova nota" });
  await dialog.getByLabel("Título").fill(title);
  await dialog.getByLabel("Conteúdo").fill(body);
  await dialog.getByRole("button", { name: "Criar" }).click();
  await expect(dialog).toBeHidden();
}

function noteLink(page: Page, title: string) {
  return page
    .getByRole("list", { name: "Lista de notas" })
    .getByRole("link", { name: new RegExp(title) });
}

test("creating a note through the dialog shows it in the list", async ({
  page,
}) => {
  const title = uniqueTitle("Compras");

  await createNote(page, title, "leite e pão");

  await expect(noteLink(page, title)).toBeVisible();
  await expect(noteLink(page, title)).toContainText("leite e pão");
});

test("a note opens on its own page", async ({ page }) => {
  const title = uniqueTitle("Ideias");
  await createNote(page, title, "uma ideia por linha");

  await noteLink(page, title).click();

  await expect(page).toHaveURL(/\/notes\/note_[0-9a-f-]{36}$/);
  await expect(
    page.getByRole("heading", { level: 1, name: title })
  ).toBeVisible();
  await expect(page.getByText("uma ideia por linha")).toBeVisible();
});

test("deleting a note after confirming removes it from the list", async ({
  page,
}) => {
  const title = uniqueTitle("Rascunho");
  await createNote(page, title, "apagar depois");
  await noteLink(page, title).click();

  await page.getByRole("button", { name: "Excluir" }).click();
  const confirm = page.getByRole("alertdialog", { name: "Excluir esta nota?" });
  await expect(confirm).toContainText(`A nota “${title}” será excluída.`);
  await confirm.getByRole("button", { name: "Excluir" }).click();

  await expect(page).toHaveURL(/\/notes$/);
  await expect(
    page.getByRole("heading", { level: 1, name: "Notas" })
  ).toBeVisible();
  await expect(noteLink(page, title)).toHaveCount(0);
});

test("an unknown note shows the not-found", async ({ page }) => {
  await page.goto(`/notes/note_${randomUUID()}`);

  await expect(
    page.getByRole("heading", { name: "Página não encontrada." })
  ).toBeVisible();
});

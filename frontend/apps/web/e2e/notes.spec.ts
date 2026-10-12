import { randomUUID } from "node:crypto";

import { expect, test } from "./fixtures";

import type { Page } from "@playwright/test";

// Against the notes service of Tilt's stack, which holds the dev data too.
// Each test creates its own note, titled with a unique `e2e-` marker, and
// asserts on it alone; the `marker` fixture deletes it afterwards.

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
  marker,
}) => {
  const title = marker("compras");

  await createNote(page, title, "leite e pão");

  await expect(noteLink(page, title)).toBeVisible();
  await expect(noteLink(page, title)).toContainText("leite e pão");
});

test("a note opens on its own page", async ({ page, marker }) => {
  const title = marker("ideias");
  await createNote(page, title, "uma ideia por linha");

  await noteLink(page, title).click();

  await expect(page).toHaveURL(
    /\/notes\/note_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/
  );
  await expect(
    page.getByRole("heading", { level: 1, name: title })
  ).toBeVisible();
  await expect(page.getByText("uma ideia por linha")).toBeVisible();
});

test("deleting a note after confirming removes it from the list", async ({
  page,
  marker,
}) => {
  const title = marker("rascunho");
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

for (const [name, id] of [
  ["an unknown note", `note_${randomUUID()}`],
  ["a note id without its prefix", randomUUID()],
  ["a note id with a malformed uuid", "note_not-a-uuid"],
]) {
  test(`${name} answers 404 with the not-found`, async ({ page }) => {
    const response = await page.goto(`/notes/${id}`);

    expect(response?.status()).toBe(404);
    await expect(
      page.getByRole("heading", { name: "Página não encontrada." })
    ).toBeVisible();
    await expect(page.getByRole("link", { name: "Boilerplate" })).toBeVisible();
  });
}

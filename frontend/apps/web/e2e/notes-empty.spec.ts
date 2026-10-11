import { expect, test } from "@playwright/test";

import { NOTES_URL } from "../playwright.config";

// Runs in its own project, before any test creates a note. The notes service
// outlives a single run in UI Mode, so the store is emptied here first.

test.beforeEach(async ({ request }) => {
  const list = await request.get(`${NOTES_URL}/notes/v1/notes`);
  expect(list.status()).toBe(200);
  const { notes } = (await list.json()) as { notes: { id: string }[] };

  for (const { id } of notes) {
    const deleted = await request.delete(`${NOTES_URL}/notes/v1/notes/${id}`);
    expect(deleted.status()).toBe(204);
  }
});

test("an empty notes list shows the empty state", async ({ page }) => {
  await page.goto("/notes");

  await expect(
    page.getByRole("heading", { level: 1, name: "Notas" })
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Nenhuma nota ainda" })
  ).toBeVisible();
  await expect(page.getByRole("list", { name: "Lista de notas" })).toHaveCount(
    0
  );
});

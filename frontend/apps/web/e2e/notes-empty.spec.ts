import { expect, test } from "@playwright/test";

// Runs in its own project, before any test creates a note: the notes service
// starts empty on every boot (see playwright.config.ts).

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

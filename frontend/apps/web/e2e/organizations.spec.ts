import { randomUUID } from "node:crypto";

import {
  MULTI_ORGANIZATION_ADMIN,
  SEED_PASSWORD,
} from "../scripts/iam-seed/users";
import { getIam } from "../shared/auth/auth";

import { signedOut } from "./auth-state";
import { expect, test } from "./fixtures";

import type { Page } from "@playwright/test";

test.use({ storageState: signedOut });

// Each test signs in on its own, so switching the active Organization never leaks into another test's session.
async function signIn(page: Page, email: string, password: string) {
  const response = await page.request.post("/api/auth/sign-in/email", {
    data: { email, password },
  });
  expect(response.status()).toBe(200);
}

function switcher(page: Page) {
  return page.getByRole("button", { name: "Trocar de Organization" });
}

async function switchTo(page: Page, organization: string) {
  await switcher(page).click();
  await page.getByRole("menuitemradio", { name: organization }).click();
  await expect(switcher(page)).toContainText(organization);
}

async function createNote(page: Page, title: string) {
  await page.getByRole("button", { name: "Nova nota" }).click();
  const dialog = page.getByRole("dialog", { name: "Nova nota" });
  await dialog.getByLabel("Título").fill(title);
  await dialog.getByLabel("Conteúdo").fill("nota de uma Organization");
  await dialog.getByRole("button", { name: "Criar" }).click();
  await expect(dialog).toBeHidden();
}

function noteLink(page: Page, title: string) {
  return page.getByRole("link", { name: new RegExp(title) });
}

async function organizationId(slug: string): Promise<string> {
  const { adapter } = await getIam().auth.$context;
  const organization = await adapter.findOne<{ id: string }>({
    model: "organization",
    where: [{ field: "slug", value: slug }],
  });
  if (organization === null) {
    throw new Error(`seed organization ${slug} is missing`);
  }
  return organization.id;
}

// Signs `page` up as a throwaway User who is a Viewer of `slugs`, oldest membership first, and deletes them afterwards.
async function withVisitorIn(
  page: Page,
  slugs: readonly string[],
  run: (userId: string) => Promise<void>
) {
  const suffix = randomUUID().slice(0, 8);
  const response = await page.request.post("/api/auth/sign-up/email", {
    data: {
      name: `Iris ${suffix}`,
      email: `e2e-organizations-${suffix}@example.test`,
      password: "e2e-password",
    },
  });
  expect(response.status()).toBe(200);
  const { user } = (await response.json()) as { user: { id: string } };

  const { auth } = getIam();
  try {
    for (const slug of slugs) {
      await auth.api.addMember({
        body: {
          organizationId: await organizationId(slug),
          userId: user.id,
          role: "viewer",
        },
      });
    }
    await run(user.id);
  } finally {
    const { internalAdapter } = await auth.$context;
    await internalAdapter.deleteUser(user.id);
  }
}

async function removeMembership(userId: string, slug: string) {
  const { adapter } = await getIam().auth.$context;
  await adapter.delete({
    model: "member",
    where: [
      { field: "userId", value: userId },
      { field: "organizationId", value: await organizationId(slug) },
    ],
  });
}

test("a Member of two Organizations switches the active one and sees its Notes, also after a reload", async ({
  page,
  marker,
}) => {
  const acmeNote = marker("acme");
  const globexNote = marker("globex");
  await signIn(page, MULTI_ORGANIZATION_ADMIN.email, SEED_PASSWORD);

  await page.goto("/notes");
  await expect(switcher(page)).toContainText("Acme");
  await createNote(page, acmeNote);
  await expect(noteLink(page, acmeNote)).toBeVisible();

  await switchTo(page, "Globex");
  await expect(noteLink(page, acmeNote)).toHaveCount(0);
  await createNote(page, globexNote);
  await expect(noteLink(page, globexNote)).toBeVisible();

  await page.reload();
  await expect(switcher(page)).toContainText("Globex");
  await expect(noteLink(page, globexNote)).toBeVisible();
  await expect(noteLink(page, acmeNote)).toHaveCount(0);

  await switchTo(page, "Acme");
  await expect(noteLink(page, acmeNote)).toBeVisible();
  await expect(noteLink(page, globexNote)).toHaveCount(0);
});

test("a User removed from the active Organization lands on another membership", async ({
  page,
}) => {
  await withVisitorIn(page, ["acme", "globex"], async (userId) => {
    await page.goto("/notes");
    await expect(switcher(page)).toContainText("Acme");

    await removeMembership(userId, "acme");
    await page.reload();

    await expect(switcher(page)).toContainText("Globex");
    await expect(
      page.getByRole("heading", { level: 1, name: "Notas" })
    ).toBeVisible();
    // Next's route announcer is an empty alert; a failed Notes load would be one with text.
    await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toHaveCount(
      0
    );
    await switcher(page).click();
    await expect(page.getByRole("menuitemradio")).toHaveText(["Globex"]);
  });
});

test("a User who lost every membership is told so and can sign out", async ({
  page,
}) => {
  await withVisitorIn(page, ["globex"], async (userId) => {
    await page.goto("/notes");
    await expect(switcher(page)).toContainText("Globex");

    await removeMembership(userId, "globex");
    await page.reload();

    await expect(
      page.getByRole("heading", {
        name: "Você não faz parte de nenhuma Organization",
      })
    ).toBeVisible();
    await expect(switcher(page)).toHaveCount(0);

    await page.getByRole("button", { name: "Sair" }).click();
    await expect(page).toHaveURL("/sign-in");
  });
});

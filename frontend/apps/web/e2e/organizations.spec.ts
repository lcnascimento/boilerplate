import { randomUUID } from "node:crypto";

import {
  ACME_OWNER,
  MULTI_ORGANIZATION_ADMIN,
  SEED_PASSWORD,
} from "../scripts/iam-seed/users";
import { getIam } from "../shared/auth/auth";

import { signedOut } from "./auth-state";
import { expect, test } from "./fixtures";

import type { Page } from "@playwright/test";

const MEMBERS_HREF = "/settings/members";
const GLOBEX_OWNER_EMAIL = "owner@globex.test";

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

function memberRow(page: Page, email: string) {
  return page.getByRole("row").filter({ hasText: email });
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

test("a Member of two Organizations switches the active one and sees its Members, also after a reload", async ({
  page,
}) => {
  await signIn(page, MULTI_ORGANIZATION_ADMIN.email, SEED_PASSWORD);

  await page.goto(MEMBERS_HREF);
  await expect(switcher(page)).toContainText("Acme");
  await expect(memberRow(page, ACME_OWNER.email)).toBeVisible();

  await switchTo(page, "Globex");
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();
  await expect(memberRow(page, ACME_OWNER.email)).toHaveCount(0);

  await page.reload();
  await expect(switcher(page)).toContainText("Globex");
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();

  await switchTo(page, "Acme");
  await expect(memberRow(page, ACME_OWNER.email)).toBeVisible();
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toHaveCount(0);
});

test("a User removed from the active Organization lands on another membership", async ({
  page,
}) => {
  await withVisitorIn(page, ["acme", "globex"], async (userId) => {
    await page.goto(MEMBERS_HREF);
    await expect(switcher(page)).toContainText("Acme");

    await removeMembership(userId, "acme");
    await page.reload();

    await expect(switcher(page)).toContainText("Globex");
    await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();
    // Next's route announcer is an empty alert; a failed page load would be one with text.
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
    await page.goto(MEMBERS_HREF);
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

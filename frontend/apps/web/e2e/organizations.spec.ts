import { randomUUID } from "node:crypto";

import { createUserOutsideSignUp } from "../scripts/iam-seed/create-user";
import {
  ACME_OWNER,
  MULTI_ORGANIZATION_ADMIN,
  SEED_PASSWORD,
} from "../scripts/iam-seed/users";
import { getIam } from "../shared/auth/auth";

import { signedOut } from "./auth-state";
import { expect, test } from "./fixtures";
import {
  openOrganizationSwitch,
  organizationMenu,
  switchOrganization,
} from "./organization-menu";

import type { Page } from "@playwright/test";

const MEMBERS_HREF = "/settings/members";
const GLOBEX_OWNER_EMAIL = "owner@globex.test";

test.use({ storageState: signedOut });

async function signIn(page: Page, email: string, password: string) {
  const response = await page.request.post("/api/auth/sign-in/email", {
    data: { email, password },
  });
  expect(response.status()).toBe(200);
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

async function asThrowawayViewerOf(
  page: Page,
  slugsOldestMembershipFirst: readonly string[],
  run: (userId: string) => Promise<void>
) {
  const suffix = randomUUID().slice(0, 8);
  const visitor = {
    name: `Iris ${suffix}`,
    email: `e2e-organizations-${suffix}@example.test`,
    password: "e2e-password",
  };
  const { auth } = getIam();
  const context = await auth.$context;
  const userId = await createUserOutsideSignUp(context, visitor);

  try {
    for (const slug of slugsOldestMembershipFirst) {
      await auth.api.addMember({
        body: {
          organizationId: await organizationId(slug),
          userId,
          role: "viewer",
        },
      });
    }
    await signIn(page, visitor.email, visitor.password);
    await run(userId);
  } finally {
    await context.internalAdapter.deleteUser(userId);
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
  await expect(organizationMenu(page)).toContainText("Acme");
  await expect(memberRow(page, ACME_OWNER.email)).toBeVisible();

  await switchOrganization(page, "Globex");
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();
  await expect(memberRow(page, ACME_OWNER.email)).toHaveCount(0);

  await page.reload();
  await expect(organizationMenu(page)).toContainText("Globex");
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();

  await switchOrganization(page, "Acme");
  await expect(memberRow(page, ACME_OWNER.email)).toBeVisible();
  await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toHaveCount(0);
});

test("a User removed from the active Organization lands on another membership", async ({
  page,
}) => {
  await asThrowawayViewerOf(page, ["acme", "globex"], async (userId) => {
    await page.goto(MEMBERS_HREF);
    await expect(organizationMenu(page)).toContainText("Acme");

    await removeMembership(userId, "acme");
    await page.reload();

    await expect(organizationMenu(page)).toContainText("Globex");
    await expect(memberRow(page, GLOBEX_OWNER_EMAIL)).toBeVisible();
    // Next's route announcer is an empty alert; a failed page load would be one with text.
    await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toHaveCount(
      0
    );
    await openOrganizationSwitch(page);
    await expect(page.getByRole("menuitemradio")).toHaveText(["Globex"]);
  });
});

test("a User who lost every membership is told so and can sign out", async ({
  page,
}) => {
  await asThrowawayViewerOf(page, ["globex"], async (userId) => {
    await page.goto(MEMBERS_HREF);
    await expect(organizationMenu(page)).toContainText("Globex");

    await removeMembership(userId, "globex");
    await page.reload();

    await expect(
      page.getByRole("heading", {
        name: "Você não faz parte de nenhuma Organization",
      })
    ).toBeVisible();
    await expect(organizationMenu(page)).toHaveCount(0);

    await page.getByRole("button", { name: "Sair" }).click();
    await expect(page).toHaveURL("/sign-in");
  });
});

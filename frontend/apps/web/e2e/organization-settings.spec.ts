import { randomUUID } from "node:crypto";

import { createUserOutsideSignUp } from "../scripts/iam-seed/create-user";
import { getIam } from "../shared/auth/auth";
import { organizationSlug } from "../shared/auth/organization-slug";

import { signedOut, storageStateOf } from "./auth-state";
import { expect, test } from "./fixtures";
import { organizationMenu } from "./organization-menu";

import type { Page } from "@playwright/test";

const SETTINGS_HREF = "/settings/organization";
const PASSWORD = "e2e-password";

function memberRow(page: Page, email: string) {
  return page.getByRole("row").filter({ hasText: email });
}

async function withThrowawayOrganization(
  run: (people: {
    owner: { name: string; email: string };
    viewer: { name: string; email: string };
  }) => Promise<void>
) {
  const suffix = randomUUID().slice(0, 8);
  const owner = {
    name: `Olga ${suffix}`,
    email: `e2e-ownership-owner-${suffix}@example.test`,
  };
  const viewer = {
    name: `Vera ${suffix}`,
    email: `e2e-ownership-viewer-${suffix}@example.test`,
  };
  const { auth } = getIam();
  const context = await auth.$context;
  const ownerId = await createUserOutsideSignUp(context, {
    ...owner,
    password: PASSWORD,
  });
  const viewerId = await createUserOutsideSignUp(context, {
    ...viewer,
    password: PASSWORD,
  });
  const name = `Ownership ${suffix}`;
  const organization = await auth.api.createOrganization({
    body: { name, slug: organizationSlug(name), userId: ownerId },
  });

  try {
    await auth.api.addMember({
      body: {
        organizationId: organization.id,
        userId: viewerId,
        role: "viewer",
      },
    });
    await run({ owner, viewer });
  } finally {
    await context.adapter.delete({
      model: "organization",
      where: [{ field: "id", value: organization.id }],
    });
    await context.internalAdapter.deleteUser(ownerId);
    await context.internalAdapter.deleteUser(viewerId);
  }
}

test.describe("as an Owner", () => {
  test.use({ storageState: signedOut });

  test("transfers the ownership to a Viewer and stays as Admin", async ({
    page,
  }) => {
    await withThrowawayOrganization(async ({ owner, viewer }) => {
      const signIn = await page.request.post("/api/auth/sign-in/email", {
        data: { email: owner.email, password: PASSWORD },
      });
      expect(signIn.status()).toBe(200);

      await page.goto("/");
      await organizationMenu(page).click();
      await page.getByRole("menuitem", { name: "Configurações" }).click();
      await expect(page).toHaveURL(SETTINGS_HREF);

      await page.getByRole("combobox", { name: "Novo Owner" }).click();
      await page
        .getByRole("option", { name: `${viewer.name} (Viewer)` })
        .click();
      await page.getByRole("button", { name: "Transferir" }).click();
      await page
        .getByRole("alertdialog", {
          name: `Transferir a propriedade para ${viewer.name}?`,
        })
        .getByRole("button", { name: "Transferir" })
        .click();

      await expect(
        page.getByText(`Propriedade transferida para ${viewer.name}.`)
      ).toBeVisible();
      await expect(
        page.getByText(
          `${viewer.name} é o Owner desta Organization. Só o Owner pode transferir a propriedade.`
        )
      ).toBeVisible();

      await page.goto("/settings/members");
      await expect(memberRow(page, viewer.email)).toContainText("Owner");
      await expect(
        memberRow(page, owner.email).getByRole("cell").nth(2)
      ).toHaveText("Admin");
      await expect(
        page.getByRole("button", { name: "Sair da Organization" })
      ).toBeEnabled();
    });
  });
});

test.describe("as the Acme Admin", () => {
  test.use({ storageState: storageStateOf("admin@acme.test") });

  test("sees who the Owner is and cannot transfer", async ({ page }) => {
    await page.goto(SETTINGS_HREF);

    await expect(
      page.getByText(
        "Ana Owner é o Owner desta Organization. Só o Owner pode transferir a propriedade."
      )
    ).toBeVisible();
    await expect(page.getByRole("combobox")).toHaveCount(0);
  });
});

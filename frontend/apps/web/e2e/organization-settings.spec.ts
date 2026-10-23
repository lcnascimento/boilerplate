import { randomUUID } from "node:crypto";

import { createUserOutsideSignUp } from "../scripts/iam-seed/create-user";
import { getIam } from "../shared/auth/auth";
import { organizationSlug } from "../shared/auth/organization-slug";
import { CREATE_OR_JOIN_HREF } from "../shared/auth/routes";

import { signedOut, storageStateOf } from "./auth-state";
import { expect, test } from "./fixtures";
import { organizationMenu } from "./organization-menu";
import { WEB_URL } from "./stack";

import type { APIRequestContext, Page } from "@playwright/test";

const SETTINGS_HREF = "/settings/organization";
const PASSWORD = "e2e-password";

interface Person {
  id: string;
  name: string;
  email: string;
}

interface ThrowawayOrganization {
  id: string;
  name: string;
  owner: Person;
  viewer: Person;
  createOrganizationFor: (owner: Person, name: string) => Promise<string>;
}

function memberRow(page: Page, email: string) {
  return page.getByRole("row").filter({ hasText: email });
}

function nameField(page: Page) {
  return page.getByRole("textbox", { name: "Nome da Organization" });
}

async function signIn(request: APIRequestContext, { email }: Person) {
  const response = await request.post("/api/auth/sign-in/email", {
    data: { email, password: PASSWORD },
  });
  expect(response.status()).toBe(200);
}

async function withThrowawayOrganization(
  run: (organization: ThrowawayOrganization) => Promise<void>
) {
  const suffix = randomUUID().slice(0, 8);
  const { auth } = getIam();
  const context = await auth.$context;
  const createdOrganizations: string[] = [];

  async function person(label: string, name: string): Promise<Person> {
    const email = `e2e-organization-settings-${label}-${suffix}@example.test`;
    const id = await createUserOutsideSignUp(context, {
      name,
      email,
      password: PASSWORD,
    });
    return { id, name, email };
  }

  async function createOrganizationFor(
    owner: Person,
    name: string
  ): Promise<string> {
    const organization = await auth.api.createOrganization({
      body: { name, slug: organizationSlug(name), userId: owner.id },
    });
    createdOrganizations.push(organization.id);
    return organization.id;
  }

  const owner = await person("owner", `Olga ${suffix}`);
  const viewer = await person("viewer", `Vera ${suffix}`);
  try {
    const name = `Settings ${suffix}`;
    const id = await createOrganizationFor(owner, name);
    await auth.api.addMember({
      body: { organizationId: id, userId: viewer.id, role: "viewer" },
    });
    await run({ id, name, owner, viewer, createOrganizationFor });
  } finally {
    for (const id of createdOrganizations) {
      await context.adapter.deleteMany({
        model: "organization",
        where: [{ field: "id", value: id }],
      });
    }
    await context.internalAdapter.deleteUser(owner.id);
    await context.internalAdapter.deleteUser(viewer.id);
  }
}

async function rowsOf(organizationId: string) {
  const { adapter } = await getIam().auth.$context;
  const where = [{ field: "organizationId", value: organizationId }];
  return {
    organization: await adapter.findOne({
      model: "organization",
      where: [{ field: "id", value: organizationId }],
    }),
    members: await adapter.count({ model: "member", where }),
    invitations: await adapter.count({ model: "invitation", where }),
  };
}

test.describe("as an Owner", () => {
  test.use({ storageState: signedOut });

  test("transfers the ownership to a Viewer and stays as Admin", async ({
    page,
  }) => {
    await withThrowawayOrganization(async ({ owner, viewer }) => {
      await signIn(page.request, owner);

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
      await expect(
        page.getByRole("button", { name: "Sair da Organization" })
      ).toBeEnabled();
      await expect(
        page.getByRole("button", { name: "Excluir Organization" })
      ).toHaveCount(0);

      await page.goto("/settings/members");
      await expect(memberRow(page, viewer.email)).toContainText("Owner");
      await expect(
        memberRow(page, owner.email).getByRole("cell").nth(2)
      ).toHaveText("Admin");
    });
  });

  test("renames the Organization and sees the new name in the Organization menu", async ({
    page,
  }) => {
    await withThrowawayOrganization(async ({ name, owner }) => {
      await signIn(page.request, owner);
      await page.goto(SETTINGS_HREF);
      await expect(organizationMenu(page)).toContainText(name);

      const renamed = `${name} renamed`;
      await nameField(page).fill(renamed);
      await page.getByRole("button", { name: "Salvar" }).click();

      await expect(page.getByText("Organization renomeada.")).toBeVisible();
      await expect(organizationMenu(page)).toContainText(renamed);
      await expect(nameField(page)).toHaveValue(renamed);
    });
  });

  test("deletes the Organization after typing its name, and its other Member lands on their next Organization", async ({
    browser,
    page,
  }) => {
    await withThrowawayOrganization(
      async ({ id, name, owner, viewer, createOrganizationFor }) => {
        const { adapter } = await getIam().auth.$context;
        await adapter.create({
          model: "invitation",
          data: {
            organizationId: id,
            email: `e2e-invitee-${randomUUID().slice(0, 8)}@example.test`,
            role: "viewer",
            status: "pending",
            expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
            createdAt: new Date(),
            inviterId: owner.id,
          },
        });
        const nextOrganization = `${name} next`;
        await createOrganizationFor(viewer, nextOrganization);

        const viewerContext = await browser.newContext({
          storageState: signedOut,
        });
        try {
          const viewerPage = await viewerContext.newPage();
          await signIn(viewerPage.request, viewer);
          await viewerPage.goto("/settings/members");
          await expect(organizationMenu(viewerPage)).toContainText(name);

          await signIn(page.request, owner);
          await page.goto(SETTINGS_HREF);
          await page
            .getByRole("button", { name: "Excluir Organization" })
            .click();
          const dialog = page.getByRole("alertdialog", {
            name: `Excluir ${name}?`,
          });
          const confirm = dialog.getByRole("button", { name: "Excluir" });
          await expect(confirm).toBeDisabled();
          await dialog
            .getByRole("textbox", { name: `Digite ${name} para confirmar` })
            .fill(name.toUpperCase());
          await expect(confirm).toBeDisabled();
          await dialog
            .getByRole("textbox", { name: `Digite ${name} para confirmar` })
            .fill(name);
          await confirm.click();

          await expect(page).toHaveURL(CREATE_OR_JOIN_HREF);
          expect(await rowsOf(id)).toEqual({
            organization: null,
            members: 0,
            invitations: 0,
          });

          await viewerPage.reload();
          await expect(organizationMenu(viewerPage)).toContainText(
            nextOrganization
          );
          await expect(
            viewerPage.getByRole("alert").filter({ hasText: /\S/ })
          ).toHaveCount(0);
        } finally {
          await viewerContext.close();
        }
      }
    );
  });
});

test.describe("as a Viewer", () => {
  test.use({ storageState: signedOut });

  test("sees the settings without rename or delete, and leaves the Organization", async ({
    page,
  }) => {
    await withThrowawayOrganization(async ({ id, name, viewer }) => {
      await signIn(page.request, viewer);
      await page.goto(SETTINGS_HREF);

      await expect(nameField(page)).toHaveValue(name);
      await expect(nameField(page)).not.toBeEditable();
      await expect(nameField(page)).toHaveAccessibleDescription(
        "Só o Owner e os Admins podem renomear a Organization."
      );
      await expect(page.getByRole("button", { name: "Salvar" })).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Excluir Organization" })
      ).toHaveCount(0);

      const rename = await page.request.post("/api/auth/organization/update", {
        data: { organizationId: id, data: { name: "hijacked" } },
        headers: { origin: WEB_URL },
      });
      expect(rename.status()).toBe(403);

      await page.getByRole("button", { name: "Sair da Organization" }).click();
      await page
        .getByRole("alertdialog", { name: "Sair da Organization?" })
        .getByRole("button", { name: "Sair" })
        .click();

      await expect(page).toHaveURL(CREATE_OR_JOIN_HREF);
      expect((await rowsOf(id)).members).toBe(1);
    });
  });
});

test.describe("as the Acme Owner", () => {
  test.use({ storageState: storageStateOf("owner@acme.test") });

  test("cannot leave before transferring the ownership, nor delete through the API", async ({
    page,
  }) => {
    await page.goto(SETTINGS_HREF);

    const leave = page.getByRole("button", { name: "Sair da Organization" });
    await expect(leave).toBeDisabled();
    await expect(leave).toHaveAccessibleDescription(
      "Você é o Owner. Transfira a propriedade a outro Member antes de sair."
    );

    const deletion = await page.request.post("/api/auth/organization/delete", {
      data: { organizationId: randomUUID() },
      headers: { origin: WEB_URL },
    });
    expect(deletion.status()).toBe(404);
  });
});

test.describe("as the Acme Admin", () => {
  test.use({ storageState: storageStateOf("admin@acme.test") });

  test("may rename but not delete, and cannot transfer", async ({ page }) => {
    await page.goto(SETTINGS_HREF);

    await expect(nameField(page)).toBeEditable();
    await expect(page.getByRole("button", { name: "Salvar" })).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Excluir Organization" })
    ).toHaveCount(0);
    await expect(
      page.getByText(
        "Ana Owner é o Owner desta Organization. Só o Owner pode transferir a propriedade."
      )
    ).toBeVisible();
    await expect(page.getByRole("combobox")).toHaveCount(0);
  });
});

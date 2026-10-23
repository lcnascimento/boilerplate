import { randomUUID } from "node:crypto";

import { createUserOutsideSignUp } from "../scripts/iam-seed/create-user";
import { getIam } from "../shared/auth/auth";
import { organizationSlug } from "../shared/auth/organization-slug";
import { SIGN_IN_HREF } from "../shared/auth/routes";

import { signedOut } from "./auth-state";
import { expect, test } from "./fixtures";
import { organizationMenu } from "./organization-menu";
import { WEB_URL } from "./stack";

import type { APIRequestContext, Page } from "@playwright/test";

const ACCOUNT_HREF = "/settings/account";
const PASSWORD = "e2e-password";

interface Person {
  id: string;
  name: string;
  email: string;
}

interface Throwaways {
  owner: Person;
  viewer: Person;
  shared: { id: string; name: string };
  createOrganizationFor: (owner: Person, name: string) => Promise<string>;
}

async function signIn(
  request: APIRequestContext,
  { email }: Person,
  password = PASSWORD
) {
  return request.post("/api/auth/sign-in/email", {
    data: { email, password },
    headers: { origin: WEB_URL },
  });
}

async function signInOk(request: APIRequestContext, person: Person) {
  expect((await signIn(request, person)).status()).toBe(200);
}

async function withThrowaways(run: (throwaways: Throwaways) => Promise<void>) {
  const suffix = randomUUID().slice(0, 8);
  const { auth } = getIam();
  const context = await auth.$context;
  const organizations: string[] = [];

  async function person(label: string, name: string): Promise<Person> {
    const email = `e2e-account-${label}-${suffix}@example.test`;
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
    organizations.push(organization.id);
    return organization.id;
  }

  const owner = await person("owner", `Olga ${suffix}`);
  const viewer = await person("viewer", `Vera ${suffix}`);
  try {
    const name = `Account ${suffix}`;
    const id = await createOrganizationFor(owner, name);
    await auth.api.addMember({
      body: { organizationId: id, userId: viewer.id, role: "viewer" },
    });
    await run({ owner, viewer, shared: { id, name }, createOrganizationFor });
  } finally {
    for (const id of organizations) {
      await context.adapter.deleteMany({
        model: "organization",
        where: [{ field: "id", value: id }],
      });
    }
    await context.internalAdapter.deleteUser(owner.id);
    await context.internalAdapter.deleteUser(viewer.id);
  }
}

async function organizationExists(id: string): Promise<boolean> {
  const { adapter } = await getIam().auth.$context;
  const organization = await adapter.findOne({
    model: "organization",
    where: [{ field: "id", value: id }],
  });
  return organization !== null;
}

function deletionBlockers(page: Page) {
  return page.getByRole("list", {
    name: "Organizations que impedem a exclusão",
  });
}

test.use({ storageState: signedOut });

test("the User opens the account settings from the User menu and renames themselves", async ({
  page,
}) => {
  await withThrowaways(async ({ viewer }) => {
    await signInOk(page.request, viewer);
    await page.goto("/");

    await page.getByRole("button", { name: "Abrir menu do usuário" }).click();
    await page.getByRole("menuitem", { name: "Configurações" }).click();
    await expect(page).toHaveURL(ACCOUNT_HREF);

    const renamed = `${viewer.name} renamed`;
    await page.getByRole("textbox", { name: "Seu nome" }).fill(renamed);
    await page.getByRole("button", { name: "Salvar" }).click();
    await expect(page.getByText("Nome atualizado.")).toBeVisible();

    await page.goto("/settings/members");
    await expect(
      page.getByRole("row").filter({ hasText: viewer.email })
    ).toContainText(renamed);
  });
});

test("changing the password refuses a wrong current one, then ends the other session and keeps this one in its Organization", async ({
  browser,
  page,
}) => {
  await withThrowaways(async ({ viewer, shared, createOrganizationFor }) => {
    const ownName = `Own ${shared.name}`;
    const own = await createOrganizationFor(viewer, ownName);
    const otherContext = await browser.newContext({ storageState: signedOut });
    try {
      const otherPage = await otherContext.newPage();
      await signInOk(otherPage.request, viewer);
      await otherPage.goto(ACCOUNT_HREF);
      await expect(otherPage).toHaveURL(ACCOUNT_HREF);

      await signInOk(page.request, viewer);
      const switched = await page.request.post(
        "/api/auth/organization/set-active",
        { data: { organizationId: own }, headers: { origin: WEB_URL } }
      );
      expect(switched.status()).toBe(200);
      await page.goto(ACCOUNT_HREF);
      await expect(organizationMenu(page)).toContainText(ownName);
      const current = page.getByLabel("Senha atual");
      const next = page.getByLabel("Nova senha");
      const submit = page.getByRole("button", { name: "Trocar senha" });

      await current.fill("not-the-password");
      await next.fill("e2e-new-password");
      await submit.click();
      await expect(
        page.getByText("A senha atual está incorreta.")
      ).toBeVisible();

      await current.fill(PASSWORD);
      await submit.click();
      await expect(
        page.getByText("Senha trocada. As outras sessões foram encerradas.")
      ).toBeVisible();

      await page.reload();
      await expect(page).toHaveURL(ACCOUNT_HREF);
      await expect(organizationMenu(page)).toContainText(ownName);
      await otherPage.reload();
      await expect(otherPage).toHaveURL(/\/sign-in/);
      expect((await signIn(page.request, viewer)).status()).toBe(401);
      expect(
        (await signIn(page.request, viewer, "e2e-new-password")).status()
      ).toBe(200);
    } finally {
      await otherContext.close();
    }
  });
});

test("deleting the account is blocked while the User owns a shared Organization, then deletes it with the sole-Member ones", async ({
  page,
}) => {
  await withThrowaways(
    async ({ owner, viewer, shared, createOrganizationFor }) => {
      const solo = await createOrganizationFor(owner, `Solo ${shared.name}`);
      await signInOk(page.request, owner);
      await page.goto(ACCOUNT_HREF);

      await expect(deletionBlockers(page).getByRole("listitem")).toHaveText([
        `${shared.name}Transferir a propriedade de ${shared.name}`,
      ]);
      await expect(
        page.getByRole("button", { name: "Excluir conta" })
      ).toBeDisabled();

      await deletionBlockers(page)
        .getByRole("link", {
          name: `Transferir a propriedade de ${shared.name}`,
        })
        .click();
      await expect(page).toHaveURL("/settings/organization");
      await expect(organizationMenu(page)).toContainText(shared.name);
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

      await page.goto(ACCOUNT_HREF);
      await expect(deletionBlockers(page)).toHaveCount(0);
      await page.getByRole("button", { name: "Excluir conta" }).click();
      const dialog = page.getByRole("alertdialog", {
        name: "Excluir a sua conta?",
      });
      const password = dialog.getByLabel("Digite a sua senha para confirmar");
      await password.fill("not-the-password");
      await dialog.getByRole("button", { name: "Excluir" }).click();
      await expect(page.getByText("A senha está incorreta.")).toBeVisible();

      await password.fill(PASSWORD);
      await dialog.getByRole("button", { name: "Excluir" }).click();

      await expect(page).toHaveURL(SIGN_IN_HREF);
      expect(await organizationExists(solo)).toBe(false);
      expect(await organizationExists(shared.id)).toBe(true);
      expect((await signIn(page.request, owner)).status()).toBe(401);

      await signInOk(page.request, viewer);
      await page.goto("/settings/members");
      await expect(
        page.getByRole("row").filter({ hasText: viewer.email })
      ).toContainText("Owner");
      await expect(
        page.getByRole("row").filter({ hasText: owner.email })
      ).toHaveCount(0);
    }
  );
});

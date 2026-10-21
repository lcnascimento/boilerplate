import { randomUUID } from "node:crypto";

import {
  expect,
  test,
  type APIRequestContext,
  type Page,
} from "@playwright/test";
import { Client } from "pg";

import { ACME_OWNER } from "../scripts/iam-seed/users";
import { SIGNED_IN_HOME } from "../shared/auth/routes";

import { signedOut } from "./auth-state";
import { IAM_DATABASE_URL } from "./stack";

interface Visitor {
  name: string;
  email: string;
  password: string;
  organizationName: string;
}

function newVisitor(): Visitor {
  const id = randomUUID().slice(0, 8);
  return {
    name: `Ada ${id}`,
    email: `e2e-sign-up-${id}@example.test`,
    password: "e2e-password",
    organizationName: `Analytical ${id}`,
  };
}

async function fillSignUp(page: Page, visitor: Visitor) {
  await page.getByLabel("Nome", { exact: true }).fill(visitor.name);
  await page.getByLabel("Email").fill(visitor.email);
  await page.getByLabel("Senha").fill(visitor.password);
  await page.getByLabel("Nome da organização").fill(visitor.organizationName);
  await page.getByRole("button", { name: "Criar conta" }).click();
}

async function signUpThroughApi(
  request: APIRequestContext,
  visitor: Visitor
): Promise<string> {
  const response = await request.post("/api/auth/sign-up/email", {
    data: visitor,
  });
  expect(response.status()).toBe(200);
  const { user } = (await response.json()) as { user: { id: string } };
  return user.id;
}

async function organizationsOf(
  request: APIRequestContext
): Promise<{ id: string; name: string }[]> {
  const response = await request.get("/api/auth/organization/list");
  expect(response.status()).toBe(200);
  return (await response.json()) as { id: string; name: string }[];
}

async function inIamDatabase(
  run: (client: Client) => Promise<unknown>
): Promise<void> {
  const client = new Client({ connectionString: IAM_DATABASE_URL });
  await client.connect();
  try {
    await run(client);
  } finally {
    await client.end();
  }
}

test.use({ storageState: signedOut });

test("a visitor signs up and lands on the home as Owner of the new organization", async ({
  page,
}) => {
  const visitor = newVisitor();
  await page.goto("/sign-in");
  await page.getByRole("link", { name: "Cadastre-se" }).click();
  await expect(page).toHaveURL("/sign-up");

  await fillSignUp(page, visitor);

  await expect(page).toHaveURL(SIGNED_IN_HOME);
  const active = await page.request.get(
    "/api/auth/organization/get-full-organization"
  );
  expect(active.status()).toBe(200);
  expect(await active.json()).toMatchObject({
    name: visitor.organizationName,
    slug: expect.stringMatching(/^analytical-[a-z0-9]+-[a-z0-9]{6}$/),
    members: [{ role: "owner", user: { email: visitor.email } }],
  });
});

test("an email already in use is flagged on its field", async ({ page }) => {
  await page.goto("/sign-up");

  await fillSignUp(page, { ...newVisitor(), email: ACME_OWNER.email });

  await expect(page.getByLabel("Email")).toHaveAttribute(
    "aria-invalid",
    "true"
  );
  await expect(
    page.getByText("Este email já está em uso. Entre ou use outro email.")
  ).toBeVisible();
  await expect(page).toHaveURL("/sign-up");
});

test("a short password is flagged on its field", async ({ page }) => {
  await page.goto("/sign-up");

  await fillSignUp(page, { ...newVisitor(), password: "short" });

  await expect(page.getByLabel("Senha")).toHaveAttribute(
    "aria-invalid",
    "true"
  );
  await expect(
    page.getByText("A senha precisa ter entre 8 e 128 caracteres.")
  ).toBeVisible();
});

test("the signed-in layout creates a missing sign-up organization once", async ({
  page,
}) => {
  const visitor = newVisitor();
  const userId = await signUpThroughApi(page.request, visitor);
  // Puts the user back where a failed sign-up hook leaves it: no organization, none active.
  await inIamDatabase(async (client) => {
    await client.query(
      `DELETE FROM organization WHERE id IN (SELECT "organizationId" FROM member WHERE "userId" = $1)`,
      [userId]
    );
    await client.query(
      `UPDATE session SET "activeOrganizationId" = NULL WHERE "userId" = $1`,
      [userId]
    );
    await client.query(
      `UPDATE "user" SET "pendingOrganizationName" = $2 WHERE id = $1`,
      [userId, visitor.organizationName]
    );
  });
  expect(await organizationsOf(page.request)).toEqual([]);

  await page.goto("/profile/preferences");
  const [created] = await organizationsOf(page.request);
  expect(created).toMatchObject({ name: visitor.organizationName });

  await page.reload();
  expect(await organizationsOf(page.request)).toEqual([created]);
  const active = await page.request.get(
    "/api/auth/organization/get-full-organization"
  );
  expect(await active.json()).toMatchObject({
    id: created?.id,
    members: [{ role: "owner", user: { email: visitor.email } }],
  });
});

test("a user who lost every membership does not get a new organization", async ({
  page,
}) => {
  const userId = await signUpThroughApi(page.request, newVisitor());
  expect(await organizationsOf(page.request)).toHaveLength(1);
  await inIamDatabase((client) =>
    client.query(`DELETE FROM member WHERE "userId" = $1`, [userId])
  );

  await page.goto("/profile/preferences");
  await page.reload();

  expect(await organizationsOf(page.request)).toEqual([]);
});

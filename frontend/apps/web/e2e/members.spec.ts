import { randomUUID } from "node:crypto";

import { expect, test, type Browser, type Page } from "@playwright/test";

import { createUserOutsideSignUp } from "../scripts/iam-seed/create-user";
import { getIam } from "../shared/auth/auth";
import { CREATE_OR_JOIN_HREF, SIGNED_IN_HOME } from "../shared/auth/routes";

import { signedOut, storageStateOf } from "./auth-state";
import { organizationMenu } from "./organization-menu";
import { WEB_URL } from "./stack";

const MEMBERS_HREF = "/settings/members";

interface Invitee {
  name: string;
  email: string;
  password: string;
}

function newInvitee(): Invitee {
  const id = randomUUID().slice(0, 8);
  return {
    name: `Gil ${id}`,
    email: `e2e-invitations-${id}@example.test`,
    password: "e2e-password",
  };
}

async function forget({ email }: Invitee) {
  const { auth } = getIam();
  const { adapter, internalAdapter } = await auth.$context;
  await adapter.deleteMany({
    model: "invitation",
    where: [{ field: "email", value: email }],
  });
  const found = await internalAdapter.findUserByEmail(email);
  if (found !== null) await internalAdapter.deleteUser(found.user.id);
}

async function openOptions(page: Page, combobox: string): Promise<string[]> {
  await page.getByRole("combobox", { name: combobox, exact: true }).click();
  const options = await page.getByRole("option").allTextContents();
  await page.keyboard.press("Escape");
  return options;
}

function rowOf(page: Page, name: string) {
  return page.getByRole("row").filter({ hasText: name });
}

function pendingInvitations(page: Page) {
  return page.getByRole("table", {
    name: "Convites pendentes da Organization",
  });
}

async function invite(page: Page, email: string) {
  await page.getByRole("button", { name: "Convidar" }).click();
  const dialog = page.getByRole("dialog", {
    name: "Convidar para a Organization",
  });
  await dialog.getByLabel("Email").fill(email);
  await dialog.getByRole("button", { name: "Convidar" }).click();
  return dialog;
}

async function asInvitee<T>(
  browser: Browser,
  run: (page: Page) => Promise<T>
): Promise<T> {
  const context = await browser.newContext({
    baseURL: WEB_URL,
    storageState: signedOut,
  });
  try {
    return await run(await context.newPage());
  } finally {
    await context.close();
  }
}

test.describe("as the Acme Owner", () => {
  test("invites an email without an account, who signs up, accepts and is then managed as a Member", async ({
    page,
    browser,
  }) => {
    const invitee = newInvitee();
    try {
      await page.goto("/");
      await organizationMenu(page).click();
      await page.getByRole("menuitem", { name: "Members" }).click();
      await expect(page).toHaveURL(MEMBERS_HREF);

      const dialog = await invite(page, invitee.email);
      await expect(dialog).toBeHidden();
      await expect(
        pendingInvitations(page)
          .getByRole("row")
          .filter({ hasText: invitee.email })
          .getByRole("cell")
          .nth(1)
      ).toHaveText("Viewer");

      await asInvitee(browser, async (inviteePage) => {
        await inviteePage.goto("/sign-up");
        await inviteePage
          .getByLabel("Nome", { exact: true })
          .fill(invitee.name);
        await inviteePage.getByLabel("Email").fill(invitee.email);
        await inviteePage.getByLabel("Senha").fill(invitee.password);
        await inviteePage.getByRole("button", { name: "Criar conta" }).click();

        await expect(inviteePage).toHaveURL(CREATE_OR_JOIN_HREF);
        await inviteePage
          .getByRole("button", { name: "Aceitar o convite de Acme" })
          .click();

        await expect(inviteePage).toHaveURL(SIGNED_IN_HOME);
        await expect(organizationMenu(inviteePage)).toContainText("Acme");
      });

      await page.reload();
      await expect(
        pendingInvitations(page).getByRole("row").filter({
          hasText: invitee.email,
        })
      ).toHaveCount(0);
      const row = rowOf(page, invitee.email);
      await expect(row.getByRole("cell").first()).toHaveText(invitee.name);
      const role = row.getByRole("combobox", {
        name: `Role de ${invitee.name}`,
      });
      await expect(role).toHaveText("Viewer");

      await role.click();
      await page.getByRole("option", { name: "Admin" }).click();
      await expect(page.getByText("Role atualizada.")).toBeVisible();
      await page.reload();
      await expect(role).toHaveText("Admin");

      await row
        .getByRole("button", { name: `Remover ${invitee.name}` })
        .click();
      await page
        .getByRole("alertdialog", { name: `Remover ${invitee.name}?` })
        .getByRole("button", { name: "Remover" })
        .click();
      await expect(row).toHaveCount(0);
    } finally {
      await forget(invitee);
    }
  });

  test("an invited User declines and the invitation leaves both lists", async ({
    page,
    browser,
  }) => {
    const invitee = newInvitee();
    try {
      await createUserOutsideSignUp(await getIam().auth.$context, invitee);
      await page.goto(MEMBERS_HREF);
      await expect(await invite(page, invitee.email)).toBeHidden();

      await asInvitee(browser, async (inviteePage) => {
        const response = await inviteePage.request.post(
          "/api/auth/sign-in/email",
          { data: { email: invitee.email, password: invitee.password } }
        );
        expect(response.status()).toBe(200);
        await inviteePage.goto("/");
        await expect(inviteePage).toHaveURL(CREATE_OR_JOIN_HREF);

        await inviteePage
          .getByRole("button", { name: "Recusar o convite de Acme" })
          .click();

        await expect(inviteePage.getByText("Convite recusado.")).toBeVisible();
        await expect(
          inviteePage.getByRole("heading", { name: "Convites recebidos" })
        ).toHaveCount(0);
        await expect(inviteePage).toHaveURL(CREATE_OR_JOIN_HREF);
      });

      await page.reload();
      await expect(rowOf(page, invitee.email)).toHaveCount(0);
    } finally {
      await forget(invitee);
    }
  });

  test("is told when the email is a Member or already invited, and cancels the invitation", async ({
    page,
  }) => {
    const invitee = newInvitee();
    try {
      await page.goto(MEMBERS_HREF);

      const dialog = await invite(page, "admin@acme.test");
      await expect(dialog.getByRole("alert")).toHaveText(
        "Esse email já é de um Member desta Organization."
      );
      await dialog.getByRole("button", { name: "Cancelar" }).click();

      await expect(await invite(page, invitee.email)).toBeHidden();
      const again = await invite(page, invitee.email);
      await expect(again.getByRole("alert")).toHaveText(
        "Esse email já tem um convite pendente para esta Organization."
      );
      await again.getByRole("button", { name: "Cancelar" }).click();

      await rowOf(page, invitee.email)
        .getByRole("button", {
          name: `Cancelar o convite de ${invitee.email}`,
        })
        .click();
      await expect(page.getByText("Convite cancelado.")).toBeVisible();
      await expect(rowOf(page, invitee.email)).toHaveCount(0);
    } finally {
      await forget(invitee);
    }
  });

  test("cannot lose the Owner Role before transferring it", async ({
    page,
  }) => {
    await page.goto(MEMBERS_HREF);

    await expect(
      page.getByRole("button", { name: "Sair da Organization" })
    ).toHaveCount(0);
    await expect(rowOf(page, "Ana Owner").getByRole("combobox")).toHaveCount(0);
    await expect(rowOf(page, "Ana Owner").getByRole("button")).toHaveCount(0);
    expect(await openOptions(page, "Role de Carla Viewer")).toEqual([
      "Admin",
      "Viewer",
    ]);

    await page.getByRole("button", { name: "Convidar" }).click();
    expect(await openOptions(page, "Role")).toEqual(["Admin", "Viewer"]);
  });

  test("is refused a second Owner by the API", async ({ page }) => {
    await page.goto(MEMBERS_HREF);
    const members = await page.request.get(
      "/api/auth/organization/list-members"
    );
    const { members: list } = (await members.json()) as {
      members: { id: string; role: string }[];
    };
    const admin = list.find(({ role }) => role === "admin");
    const owner = list.find(({ role }) => role === "owner");
    if (admin === undefined || owner === undefined) {
      throw new Error("the Acme seed has no Admin or no Owner");
    }

    for (const [memberId, role, code] of [
      [admin.id, "owner", "OWNER_NOT_ASSIGNABLE"],
      [owner.id, "admin", "YOU_CANNOT_LEAVE_THE_ORGANIZATION_WITHOUT_AN_OWNER"],
    ] as const) {
      const response = await page.request.post(
        "/api/auth/organization/update-member-role",
        { data: { memberId, role }, headers: { origin: WEB_URL } }
      );
      expect(response.status()).toBe(400);
      expect(((await response.json()) as { code: string }).code).toBe(code);
    }
  });
});

test.describe("as the Acme Admin", () => {
  test.use({ storageState: storageStateOf("admin@acme.test") });

  test("never sees the Owner Role", async ({ page }) => {
    await page.goto(MEMBERS_HREF);

    await expect(rowOf(page, "Ana Owner").getByRole("combobox")).toHaveCount(0);
    await expect(rowOf(page, "Ana Owner").getByRole("button")).toHaveCount(0);
    expect(await openOptions(page, "Role de Carla Viewer")).toEqual([
      "Admin",
      "Viewer",
    ]);

    await page.getByRole("button", { name: "Convidar" }).click();
    expect(await openOptions(page, "Role")).toEqual(["Admin", "Viewer"]);
  });
});

test.describe("as the Acme Viewer", () => {
  test.use({ storageState: storageStateOf("viewer@acme.test") });

  test("sees the Members without invite, Role or remove actions", async ({
    page,
  }) => {
    await page.goto(MEMBERS_HREF);

    const table = page.getByRole("table", { name: "Members da Organization" });
    for (const [name, email, role] of [
      ["Ana Owner", "owner@acme.test", "Owner"],
      ["Bruno Admin", "admin@acme.test", "Admin"],
      ["Carla Viewer (você)", "viewer@acme.test", "Viewer"],
    ] as const) {
      await expect(
        table.getByRole("row").filter({ hasText: email }).getByRole("cell")
      ).toHaveText([name, email, role, ""]);
    }
    await expect(page.getByRole("button", { name: "Convidar" })).toHaveCount(0);
    await expect(table.getByRole("combobox")).toHaveCount(0);
    await expect(table.getByRole("button")).toHaveCount(0);
  });
});

import { expect, test, type Page } from "@playwright/test";

import { storageStateOf } from "./auth-state";
import { organizationMenu } from "./organization-menu";
import { WEB_URL } from "./stack";

const MEMBERS_HREF = "/settings/members";
const OUTSIDER = { name: "Diego Owner", email: "owner@globex.test" };

async function openOptions(page: Page, combobox: string): Promise<string[]> {
  await page.getByRole("combobox", { name: combobox, exact: true }).click();
  const options = await page.getByRole("option").allTextContents();
  await page.keyboard.press("Escape");
  return options;
}

function rowOf(page: Page, name: string) {
  return page.getByRole("row").filter({ hasText: name });
}

async function restoreSeed(page: Page) {
  const response = await page.request.post(
    "/api/auth/organization/remove-member",
    {
      data: { memberIdOrEmail: OUTSIDER.email },
      headers: { origin: WEB_URL },
    }
  );
  expect([200, 400]).toContain(response.status());
}

test.describe("as the Acme Owner", () => {
  test.describe("managing a Member", () => {
    test.beforeEach(async ({ page }) => restoreSeed(page));
    test.afterEach(async ({ page }) => restoreSeed(page));

    test("adds a seed User as Viewer, changes the Role and removes them", async ({
      page,
    }) => {
      await page.goto("/");
      await organizationMenu(page).click();
      await page.getByRole("menuitem", { name: "Members" }).click();
      await expect(page).toHaveURL(MEMBERS_HREF);

      await page.getByRole("button", { name: "Adicionar Member" }).click();
      const dialog = page.getByRole("dialog", { name: "Adicionar Member" });
      await dialog.getByLabel("Email").fill(OUTSIDER.email);
      await expect(
        dialog.getByRole("combobox", { name: "Role", exact: true })
      ).toHaveText("Viewer");
      await dialog.getByRole("button", { name: "Adicionar" }).click();

      await expect(dialog).toBeHidden();
      const row = rowOf(page, OUTSIDER.email);
      await expect(row.getByRole("cell").first()).toHaveText(OUTSIDER.name);
      const role = row.getByRole("combobox", {
        name: `Role de ${OUTSIDER.name}`,
      });
      await expect(role).toHaveText("Viewer");

      await role.click();
      await page.getByRole("option", { name: "Admin" }).click();
      await expect(page.getByText("Role atualizada.")).toBeVisible();
      await page.reload();
      await expect(role).toHaveText("Admin");

      await row
        .getByRole("button", { name: `Remover ${OUTSIDER.name}` })
        .click();
      await page
        .getByRole("alertdialog", { name: `Remover ${OUTSIDER.name}?` })
        .getByRole("button", { name: "Remover" })
        .click();
      await expect(row).toHaveCount(0);
    });
  });

  test("tells an unknown email apart from an existing Member", async ({
    page,
  }) => {
    await page.goto(MEMBERS_HREF);
    await page.getByRole("button", { name: "Adicionar Member" }).click();
    const dialog = page.getByRole("dialog", { name: "Adicionar Member" });

    await dialog.getByLabel("Email").fill("nobody@acme.test");
    await dialog.getByRole("button", { name: "Adicionar" }).click();
    await expect(dialog.getByRole("alert")).toHaveText(
      "Nenhum User cadastrado com esse email."
    );

    await dialog.getByLabel("Email").fill("admin@acme.test");
    await dialog.getByRole("button", { name: "Adicionar" }).click();
    await expect(dialog.getByRole("alert")).toHaveText(
      "Esse User já é Member desta Organization."
    );
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

    await page.getByRole("button", { name: "Adicionar Member" }).click();
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

    await page.getByRole("button", { name: "Adicionar Member" }).click();
    expect(await openOptions(page, "Role")).toEqual(["Admin", "Viewer"]);
  });
});

test.describe("as the Acme Viewer", () => {
  test.use({ storageState: storageStateOf("viewer@acme.test") });

  test("sees the Members without add, Role or remove actions", async ({
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
    await expect(
      page.getByRole("button", { name: "Adicionar Member" })
    ).toHaveCount(0);
    await expect(table.getByRole("combobox")).toHaveCount(0);
    await expect(table.getByRole("button")).toHaveCount(0);
  });
});

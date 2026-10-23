import { expect, type Page } from "@playwright/test";

export function organizationMenu(page: Page) {
  return page.getByRole("button", { name: "Abrir menu da Organization" });
}

export async function openOrganizationSwitch(page: Page) {
  await organizationMenu(page).click();
  await page.getByRole("menuitem", { name: "Trocar Organization" }).click();
}

export async function switchOrganization(page: Page, organization: string) {
  await openOrganizationSwitch(page);
  await page.getByRole("menuitemradio", { name: organization }).click();
  await expect(organizationMenu(page)).toContainText(organization);
}

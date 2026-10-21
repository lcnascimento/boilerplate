import { expect, test as setup } from "@playwright/test";

import { SEED_PASSWORD, seedUsers } from "../scripts/iam-seed/users";

import { storageStateOf } from "./auth-state";

for (const { email } of seedUsers) {
  setup(`sign in as ${email}`, async ({ request }) => {
    const response = await request.post("/api/auth/sign-in/email", {
      data: { email, password: SEED_PASSWORD },
    });
    expect(response.status()).toBe(200);

    await request.storageState({ path: storageStateOf(email) });
  });
}

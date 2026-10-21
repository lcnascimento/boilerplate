import { randomUUID } from "node:crypto";

import { test as base, expect, type APIRequestContext } from "@playwright/test";

// example:begin
import { noteListSchema } from "@boilerplate/api/notes/schemas";

import { ACME_OWNER } from "../scripts/iam-seed/users";
import { mintActorToken } from "../shared/auth/actor-token";
import { getIam } from "../shared/auth/auth";

import { NOTES_URL } from "./stack";
// example:end

type Marker = (label: string) => string;

interface CleanupArgs {
  request: APIRequestContext;
  markers: readonly string[];
}

type Cleanup = (args: CleanupArgs) => Promise<void>;

// example:begin
async function acmeOwnerToken(): Promise<string> {
  const context = await getIam().auth.$context;
  const owner = await context.internalAdapter.findUserByEmail(ACME_OWNER.email);
  const acme = await context.adapter.findOne<{ id: string }>({
    model: "organization",
    where: [{ field: "slug", value: "acme" }],
  });
  if (owner === null || acme === null) {
    throw new Error(
      "the iam seed has no Acme owner: run the iam-seed resource"
    );
  }
  return mintActorToken({ userId: owner.user.id, organizationId: acme.id });
}

async function deleteMarkedNotes({ request, markers }: CleanupArgs) {
  const headers = { Authorization: `Bearer ${await acmeOwnerToken()}` };
  const list = await request.get(`${NOTES_URL}/notes/v1/notes`, { headers });
  expect(list.status()).toBe(200);
  const notes = noteListSchema.parse(await list.json());

  for (const { id, title } of notes) {
    if (!markers.some((marker) => title.includes(marker))) continue;
    const deleted = await request.delete(`${NOTES_URL}/notes/v1/notes/${id}`, {
      headers,
    });
    expect(deleted.status()).toBe(204);
  }
}
// example:end

const cleanups: readonly Cleanup[] = [
  // example:begin
  deleteMarkedNotes,
  // example:end
];

export const test = base.extend<{ marker: Marker }>({
  marker: async ({ request }, use) => {
    const markers: string[] = [];
    await use((label) => {
      const marker = `e2e-${label}-${randomUUID().slice(0, 8)}`;
      markers.push(marker);
      return marker;
    });

    if (markers.length === 0) return;
    for (const cleanup of cleanups) await cleanup({ request, markers });
  },
});

export { expect };

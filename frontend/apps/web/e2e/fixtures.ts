import { randomUUID } from "node:crypto";

import { test as base, expect, type APIRequestContext } from "@playwright/test";

// example:begin
import { noteListSchema } from "@boilerplate/api/notes/schemas";

import {
  seedOrganizations,
  type SeedOrganization,
} from "../scripts/iam-seed/users";
import { findActor } from "../shared/auth/actor";
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
async function ownerToken({ slug, owner }: SeedOrganization): Promise<string> {
  const { auth, database } = getIam();
  const context = await auth.$context;
  const user = await context.internalAdapter.findUserByEmail(owner.email);
  const organization = await context.adapter.findOne<{ id: string }>({
    model: "organization",
    where: [{ field: "slug", value: slug }],
  });
  if (user === null || organization === null) {
    throw new Error(
      `the iam seed has no ${slug} owner: run the iam-seed resource`
    );
  }
  return mintActorToken(
    await findActor(database, {
      userId: user.user.id,
      organizationId: organization.id,
    })
  );
}

async function deleteMarkedNotes({ request, markers }: CleanupArgs) {
  for (const organization of seedOrganizations) {
    const headers = {
      Authorization: `Bearer ${await ownerToken(organization)}`,
    };
    const list = await request.get(`${NOTES_URL}/notes/v1/notes`, { headers });
    expect(list.status()).toBe(200);
    const notes = noteListSchema.parse(await list.json());

    for (const { id, title } of notes) {
      if (!markers.some((marker) => title.includes(marker))) continue;
      const deleted = await request.delete(
        `${NOTES_URL}/notes/v1/notes/${id}`,
        { headers }
      );
      expect(deleted.status()).toBe(204);
    }
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

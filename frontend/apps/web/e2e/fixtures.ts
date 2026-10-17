import { randomUUID } from "node:crypto";

import { test as base, expect, type APIRequestContext } from "@playwright/test";

// example:begin
import { noteListSchema } from "@boilerplate/api/notes/schemas";

import { NOTES_URL } from "./stack";
// example:end

type Marker = (label: string) => string;

interface CleanupArgs {
  request: APIRequestContext;
  markers: readonly string[];
}

type Cleanup = (args: CleanupArgs) => Promise<void>;

// example:begin
async function deleteMarkedNotes({ request, markers }: CleanupArgs) {
  const list = await request.get(`${NOTES_URL}/notes/v1/notes`);
  expect(list.status()).toBe(200);
  const notes = noteListSchema.parse(await list.json());

  for (const { id, title } of notes) {
    if (!markers.some((marker) => title.includes(marker))) continue;
    const deleted = await request.delete(`${NOTES_URL}/notes/v1/notes/${id}`);
    expect(deleted.status()).toBe(204);
  }
}
// example:end

const cleanups: readonly Cleanup[] = [
  // One per service whose data the e2e creates.
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

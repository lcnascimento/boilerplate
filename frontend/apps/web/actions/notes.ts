"use server";

import { notFound, redirect } from "next/navigation";

import { invalidInputError, type Result } from "@boilerplate/api/core";
import {
  createNoteInputSchema,
  createNotesClient,
  noteIdSchema,
  parseNotesEnv,
  type Note,
} from "@boilerplate/api/notes";

import { mintActorToken } from "@/shared/auth/actor-token";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { logAPIFailure } from "@/shared/logging";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

async function requestActorToken(): Promise<string> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind === "no-organization") redirect("/");
  return mintActorToken(requestActor.actor);
}

function notesClient() {
  return createNotesClient({
    baseUrl: parseNotesEnv(process.env).NOTES_API_URL,
    getToken: requestActorToken,
  });
}

export async function listNotes(): Promise<Result<Note[]>> {
  return logAPIFailure("notes.list", notesClient().list());
}

export async function getNote(id: unknown): Promise<Result<Note>> {
  const noteId = noteIdSchema.safeParse(id);
  if (!noteId.success) notFound();

  const result = await logAPIFailure(
    "notes.get",
    notesClient().get(noteId.data)
  );
  if (result.kind === "error" && result.error.status === 404) notFound();
  return result;
}

export async function createNote(input: unknown): Promise<Result<Note>> {
  const parsed = createNoteInputSchema.safeParse(input);
  if (!parsed.success) {
    return { kind: "error", error: invalidInputError(parsed.error) };
  }
  return logAPIFailure("notes.create", notesClient().create(parsed.data));
}

export async function deleteNote(id: unknown): Promise<Result<undefined>> {
  const noteId = noteIdSchema.safeParse(id);
  if (!noteId.success) {
    return { kind: "error", error: invalidInputError(noteId.error) };
  }
  return logAPIFailure("notes.delete", notesClient().delete(noteId.data));
}

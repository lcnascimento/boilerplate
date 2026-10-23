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

import { serverAction } from "./server-action";

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

export const getNote = serverAction(
  noteIdSchema,
  async (noteId): Promise<Result<Note>> => {
    const result = await logAPIFailure("notes.get", notesClient().get(noteId));
    if (result.kind === "error" && result.error.status === 404) notFound();
    return result;
  },
  () => notFound()
);

export const createNote = serverAction(
  createNoteInputSchema,
  (input): Promise<Result<Note>> =>
    logAPIFailure("notes.create", notesClient().create(input)),
  (error): Result<Note> => ({ kind: "error", error: invalidInputError(error) })
);

export const deleteNote = serverAction(
  noteIdSchema,
  (noteId): Promise<Result<undefined>> =>
    logAPIFailure("notes.delete", notesClient().delete(noteId)),
  (error): Result<undefined> => ({
    kind: "error",
    error: invalidInputError(error),
  })
);

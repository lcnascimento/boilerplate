"use server";

import { notFound } from "next/navigation";

import { invalidInputError, type Result } from "@boilerplate/api/core";
import {
  createNoteInputSchema,
  createNotesClient,
  noteIdSchema,
  parseNotesEnv,
  type Note,
} from "@boilerplate/api/notes";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

function notesClient() {
  return createNotesClient({
    baseUrl: parseNotesEnv(process.env).NOTES_API_URL,
  });
}

export async function listNotes(): Promise<Result<Note[]>> {
  return notesClient().list();
}

export async function getNote(id: unknown): Promise<Result<Note>> {
  const noteId = noteIdSchema.safeParse(id);
  if (!noteId.success) notFound();

  const result = await notesClient().get(noteId.data);
  if (result.kind === "error" && result.error.status === 404) notFound();
  return result;
}

export async function createNote(input: unknown): Promise<Result<Note>> {
  const parsed = createNoteInputSchema.safeParse(input);
  if (!parsed.success) {
    return { kind: "error", error: invalidInputError(parsed.error) };
  }
  return notesClient().create(parsed.data);
}

export async function deleteNote(id: unknown): Promise<Result<undefined>> {
  const noteId = noteIdSchema.safeParse(id);
  if (!noteId.success) {
    return { kind: "error", error: invalidInputError(noteId.error) };
  }
  return notesClient().delete(noteId.data);
}

"use server";

import { notFound } from "next/navigation";

import {
  createNoteInputSchema,
  createNotesClient,
  noteIdSchema,
  parseNotesEnv,
  type Note,
} from "@boilerplate/api/notes";

import type { APIError, Result } from "@boilerplate/api/core";

// Every function here is a public endpoint: arguments arrive as `unknown` and
// are parsed before reaching the notes service.

function notesClient() {
  return createNotesClient({
    baseUrl: parseNotesEnv(process.env).NOTES_API_URL,
  });
}

function invalidInput(reasons: string[]): { kind: "error"; error: APIError } {
  return {
    kind: "error",
    error: {
      code: "INVALID_INPUT",
      message: "invalid input",
      retryable: false,
      details: { reasons },
      status: 400,
    },
  };
}

export async function listNotes(): Promise<Result<Note[]>> {
  return notesClient().list();
}

/** The note, or the route's not-found when the id is malformed or unknown. */
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
    return invalidInput(
      parsed.error.issues.map(
        (issue) => `[${issue.path.join(".")}] ${issue.message}`
      )
    );
  }
  return notesClient().create(parsed.data);
}

export async function deleteNote(id: unknown): Promise<Result<undefined>> {
  const noteId = noteIdSchema.safeParse(id);
  if (!noteId.success) return invalidInput(["[id] invalid note id"]);
  return notesClient().delete(noteId.data);
}

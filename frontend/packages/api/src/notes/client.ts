import { z } from "zod";

import { invalidInputError, request, type Result } from "../core";

import {
  createNoteInputSchema,
  noteListSchema,
  noteSchema,
  type CreateNoteInput,
  type Note,
  type NoteId,
} from "./schemas";

interface NotesClientOptions {
  baseUrl: string;
}

export interface NotesClient {
  create(input: CreateNoteInput): Promise<Result<Note>>;
  list(): Promise<Result<Note[]>>;
  get(id: NoteId): Promise<Result<Note>>;
  delete(id: NoteId): Promise<Result<undefined>>;
}

function toWireCreateNote({ title, body }: CreateNoteInput) {
  return body === undefined ? { title } : { title, body };
}

export function createNotesClient({
  baseUrl,
}: NotesClientOptions): NotesClient {
  const notesUrl = `${baseUrl}/notes/v1/notes`;
  const noteUrl = (id: NoteId) => `${notesUrl}/${encodeURIComponent(id)}`;

  return {
    create: async (input) => {
      const parsed = createNoteInputSchema.safeParse(input);
      if (!parsed.success) {
        return { kind: "error", error: invalidInputError(parsed.error) };
      }
      return request({
        url: notesUrl,
        method: "POST",
        body: toWireCreateNote(parsed.data),
        schema: noteSchema,
      });
    },
    list: () => request({ url: notesUrl, schema: noteListSchema }),
    get: (id) => request({ url: noteUrl(id), schema: noteSchema }),
    delete: (id) =>
      request({ url: noteUrl(id), method: "DELETE", schema: z.undefined() }),
  };
}

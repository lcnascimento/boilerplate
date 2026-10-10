import { z } from "zod";

import { request, type Result } from "../core";

import {
  noteListSchema,
  noteSchema,
  type CreateNoteInput,
  type Note,
  type NoteId,
} from "./schemas";

interface NotesClientOptions {
  /** The notes service root, e.g. `http://localhost:8080`. */
  baseUrl: string;
}

export interface NotesClient {
  create(input: CreateNoteInput): Promise<Result<Note>>;
  list(): Promise<Result<Note[]>>;
  get(id: NoteId): Promise<Result<Note>>;
  delete(id: NoteId): Promise<Result<undefined>>;
}

/** The domain input in the wire shape the service reads. */
function toWireCreateNote({ title, body }: CreateNoteInput) {
  return body === undefined ? { title } : { title, body };
}

export function createNotesClient({
  baseUrl,
}: NotesClientOptions): NotesClient {
  const notesUrl = `${baseUrl}/notes/v1/notes`;
  const noteUrl = (id: NoteId) => `${notesUrl}/${encodeURIComponent(id)}`;

  return {
    create: (input) =>
      request({
        url: notesUrl,
        method: "POST",
        body: toWireCreateNote(input),
        schema: noteSchema,
      }),
    list: () => request({ url: notesUrl, schema: noteListSchema }),
    get: (id) => request({ url: noteUrl(id), schema: noteSchema }),
    delete: (id) =>
      request({ url: noteUrl(id), method: "DELETE", schema: z.undefined() }),
  };
}

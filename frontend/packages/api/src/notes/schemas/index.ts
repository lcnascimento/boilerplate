import { z } from "zod";

export const NOTE_TITLE_MAX_LENGTH = 120;
export const NOTE_BODY_MAX_LENGTH = 10_000;

export const noteIdSchema = z
  .string()
  .regex(/^note_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i)
  .brand<"NoteId">();

export type NoteId = z.infer<typeof noteIdSchema>;

export const noteSchema = z
  .object({
    id: noteIdSchema,
    title: z.string(),
    body: z.string(),
    created_at: z.iso.datetime({ offset: true }),
  })
  .transform(({ id, title, body, created_at }) => ({
    id,
    title,
    body,
    createdAt: new Date(created_at),
  }));

export type Note = z.infer<typeof noteSchema>;

export const noteListSchema = z
  .object({ notes: z.array(noteSchema) })
  .transform(({ notes }) => notes);

export const createNoteInputSchema = z.object({
  title: z.string().trim().min(1).max(NOTE_TITLE_MAX_LENGTH),
  body: z.string().max(NOTE_BODY_MAX_LENGTH).optional(),
});

export type CreateNoteInput = z.infer<typeof createNoteInputSchema>;

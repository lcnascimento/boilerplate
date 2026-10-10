import { z } from "zod";

const notesEnvSchema = z.object({
  NOTES_API_URL: z
    .url({
      protocol: /^https?$/,
      error: (issue) =>
        issue.input === undefined
          ? "NOTES_API_URL is required"
          : "NOTES_API_URL must be an http(s) URL",
    })
    .transform((url) => url.replace(/\/+$/, "")),
});

export type NotesEnv = z.infer<typeof notesEnvSchema>;

/**
 * Reads the notes service settings from the environment (pass `process.env`)
 * and throws when they are missing or invalid. Call it at server start so a
 * misconfigured app fails before serving, not on its first request.
 */
export function parseNotesEnv(
  env: Record<string, string | undefined>
): NotesEnv {
  const parsed = notesEnvSchema.safeParse(env);
  if (!parsed.success) {
    const reasons = parsed.error.issues.map((issue) => issue.message);
    throw new Error(`Invalid environment: ${reasons.join("; ")}`);
  }
  return parsed.data;
}

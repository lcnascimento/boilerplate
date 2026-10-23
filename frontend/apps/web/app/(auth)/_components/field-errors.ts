import type { z } from "zod";

export type FieldErrors<Field extends string> = Partial<Record<Field, string>>;

export function firstErrorPerField<Field extends string>(
  error: z.ZodError,
  fieldSchema: z.ZodEnum<Record<Field, Field>>,
  message: (field: Field, code: string) => string
): FieldErrors<Field> {
  const errors: FieldErrors<Field> = {};
  for (const issue of error.issues) {
    const field = fieldSchema.safeParse(issue.path[0]);
    if (!field.success || errors[field.data] !== undefined) continue;
    errors[field.data] = message(field.data, issue.code);
  }
  return errors;
}

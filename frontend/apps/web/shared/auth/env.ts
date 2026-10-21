import { z } from "zod";

const iamEnvSchema = z.object({
  IAM_DATABASE_URL: z.url({
    protocol: /^postgres(ql)?$/,
    error: (issue) =>
      issue.input === undefined
        ? "IAM_DATABASE_URL is required"
        : "IAM_DATABASE_URL must be a postgres:// URL",
  }),
  BETTER_AUTH_SECRET: z
    .string({ error: "BETTER_AUTH_SECRET is required" })
    .min(32, "BETTER_AUTH_SECRET must have at least 32 characters"),
  BETTER_AUTH_URL: z
    .url({
      protocol: /^https?$/,
      error: (issue) =>
        issue.input === undefined
          ? "BETTER_AUTH_URL is required"
          : "BETTER_AUTH_URL must be an http(s) URL",
    })
    .transform((url) => url.replace(/\/+$/, "")),
});

export type IamEnv = z.infer<typeof iamEnvSchema>;

export function parseIamEnv(env: Record<string, string | undefined>): IamEnv {
  const parsed = iamEnvSchema.safeParse(env);
  if (!parsed.success) {
    const reasons = parsed.error.issues.map((issue) => issue.message);
    throw new Error(`Invalid environment: ${reasons.join("; ")}`);
  }
  return parsed.data;
}

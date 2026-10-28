import { z } from "zod";

const DEV_ONLY_PREFIX = "dev-only-";
const DEV_CREDENTIALS = ["BETTER_AUTH_SECRET", "AUDIT_IP_HASH_KEY"] as const;

const iamEnvSchema = z
  .object({
    NODE_ENV: z.string().optional(),
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
    AUDIT_IP_HASH_KEY: z
      .string({ error: "AUDIT_IP_HASH_KEY is required" })
      .min(32, "AUDIT_IP_HASH_KEY must have at least 32 characters"),
  })
  .superRefine((env, ctx) => {
    if (env.NODE_ENV !== "production") return;
    for (const key of DEV_CREDENTIALS) {
      if (env[key].startsWith(DEV_ONLY_PREFIX)) {
        ctx.addIssue({
          code: "custom",
          path: [key],
          message: `${key} is a dev-only value; generate one for production (openssl rand -base64 32)`,
        });
      }
    }
  })
  .transform(({ NODE_ENV: _, ...env }) => env);

export type IamEnv = z.infer<typeof iamEnvSchema>;

export function parseIamEnv(env: Record<string, string | undefined>): IamEnv {
  const parsed = iamEnvSchema.safeParse(env);
  if (!parsed.success) {
    const reasons = parsed.error.issues.map((issue) => issue.message);
    throw new Error(`Invalid environment: ${reasons.join("; ")}`);
  }
  return parsed.data;
}

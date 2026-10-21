import { faro } from "@grafana/faro-web-sdk";
import { z } from "zod";

const faroEnvSchema = z.object({
  NEXT_PUBLIC_FARO_URL: z.url().optional(),
  NEXT_PUBLIC_FARO_SAMPLE_RATE: z.coerce.number().min(0).max(1).default(1),
});

export type FaroEnv =
  { kind: "disabled" } | { kind: "enabled"; url: string; sampleRate: number };

export function parseFaroEnv(env: Record<string, string | undefined>): FaroEnv {
  const blankAsUnset = Object.fromEntries(
    Object.entries(env).filter(
      ([, value]) => value !== undefined && value !== ""
    )
  );
  const parsed = faroEnvSchema.parse(blankAsUnset);

  if (parsed.NEXT_PUBLIC_FARO_URL === undefined) return { kind: "disabled" };

  return {
    kind: "enabled",
    url: parsed.NEXT_PUBLIC_FARO_URL,
    sampleRate: parsed.NEXT_PUBLIC_FARO_SAMPLE_RATE,
  };
}

// Before initializeFaro, and whenever NEXT_PUBLIC_FARO_URL is unset, faro.api is Faro's no-op API.
export function reportBoundaryError(error: Error & { digest?: string }): void {
  faro.api.pushError(
    error,
    error.digest === undefined
      ? undefined
      : { context: { digest: error.digest } }
  );
}

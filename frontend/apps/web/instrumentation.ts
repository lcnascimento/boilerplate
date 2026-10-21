import type { Instrumentation } from "next";

export async function register(): Promise<void> {
  if (process.env.NEXT_RUNTIME !== "nodejs") return;

  await import("./instrumentation.node");
  const { logger } = await import("@/shared/logging");

  await Promise.all([
    // example:begin
    import("@boilerplate/api/notes").then(({ parseNotesEnv }) =>
      parseNotesEnv(process.env)
    ),
    // example:end
  ]);

  logger.info("server started");
}

export const onRequestError: Instrumentation.onRequestError = async (
  ...args
) => {
  if (process.env.NEXT_RUNTIME !== "nodejs") return;

  const { recordRequestError } = await import("@/shared/logging");
  await recordRequestError(...args);
};

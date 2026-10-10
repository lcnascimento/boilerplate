/**
 * Runs once when the server starts, before it serves a request. Each backend
 * service validates its env here, so a missing `<NAME>_API_URL` stops the
 * server at boot with a clear error instead of failing the first request.
 */
export async function register(): Promise<void> {
  if (process.env.NEXT_RUNTIME !== "nodejs") return;

  await Promise.all([
    // example:begin
    import("@boilerplate/api/notes").then(({ parseNotesEnv }) =>
      parseNotesEnv(process.env)
    ),
    // example:end
  ]);
}

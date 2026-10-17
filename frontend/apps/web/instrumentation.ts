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

export const WEB_URL = process.env.WEB_URL ?? "http://localhost:3000";
// example:begin
export const NOTES_URL = process.env.NOTES_URL ?? "http://localhost:8080";
// example:end

export const stackProbes: readonly { name: string; url: string }[] = [
  { name: "web", url: `${WEB_URL}/` },
  // example:begin
  { name: "notes", url: `${NOTES_URL}/health` },
  // example:end
];

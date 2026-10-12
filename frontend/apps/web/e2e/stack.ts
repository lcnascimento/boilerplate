// The e2e suite runs against Tilt's stack, already up: the web app in dev mode
// and each backend service. Each URL comes from env, defaulting to Tilt's port.
export const WEB_URL = process.env.WEB_URL ?? "http://localhost:3000";
// example:begin
export const NOTES_URL = process.env.NOTES_URL ?? "http://localhost:8080";
// example:end

/** What the global setup probes before any test runs: one entry per server. */
export const stackProbes: readonly { name: string; url: string }[] = [
  { name: "web", url: `${WEB_URL}/` },
  // example:begin
  { name: "notes", url: `${NOTES_URL}/health` },
  // example:end
];

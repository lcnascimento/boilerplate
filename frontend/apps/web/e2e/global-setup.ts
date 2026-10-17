import { stackProbes } from "./stack";

// Generous: Tilt marks `web` ready once `next dev` binds its port, before it
// compiles a route, so on a cold cache the first `GET /` waits for that compile.
// A stack that is down refuses the connection at once.
const PROBE_TIMEOUT_MS = 60_000;

async function probe(url: string): Promise<string | undefined> {
  try {
    const response = await fetch(url, {
      signal: AbortSignal.timeout(PROBE_TIMEOUT_MS),
    });
    return response.ok ? undefined : `answered ${response.status}`;
  } catch (error) {
    // `cause.code` (e.g. ECONNREFUSED) says more than fetch's "fetch failed".
    const cause = error instanceof Error ? error.cause : undefined;
    if (cause instanceof Error && "code" in cause) return String(cause.code);
    return error instanceof Error ? error.message : String(error);
  }
}

export default async function globalSetup() {
  const failures = (
    await Promise.all(
      stackProbes.map(async ({ name, url }) => {
        const failure = await probe(url);
        return failure === undefined
          ? undefined
          : `  - ${name} (${url}): ${failure}`;
      })
    )
  ).filter((failure) => failure !== undefined);

  if (failures.length > 0) {
    throw new Error(
      [
        "The e2e suite runs against Tilt's stack, and it is not up:",
        ...failures,
        "Start it with `tilt up` (or `TILT_STACK=minimal tilt up`) and wait for its resources to be ready.",
      ].join("\n")
    );
  }
}

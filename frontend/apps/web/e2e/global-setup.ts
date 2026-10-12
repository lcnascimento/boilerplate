import { stackProbes } from "./stack";

// Fails the run before any browser starts when a server of the stack does not
// answer: the suite never starts servers of its own. A stack that is down
// refuses the connection at once, so the timeout only bounds a slow answer.
// It is generous because Tilt marks `web` ready once `next dev` binds its
// port, before it compiles any route: on a cold cache (a fresh CI runner) the
// first `GET /` waits for that compile, which takes several seconds.
const PROBE_TIMEOUT_MS = 60_000;

async function probe(url: string): Promise<string | undefined> {
  try {
    const response = await fetch(url, {
      signal: AbortSignal.timeout(PROBE_TIMEOUT_MS),
    });
    return response.ok ? undefined : `answered ${response.status}`;
  } catch (error) {
    // fetch wraps the network error in `cause`, whose `code` (e.g.
    // ECONNREFUSED) says more than its "fetch failed" message.
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

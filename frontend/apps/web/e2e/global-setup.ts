import { stackProbes } from "./stack";

// Tilt marks `web` ready before `next dev` compiles a route, so a cold first `GET /` waits for the compile.
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

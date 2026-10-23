import "server-only";

import { headers } from "next/headers";
import { cache } from "react";

import { getIam } from "./auth";

export const getRequestSession = cache(async () =>
  getIam().auth.api.getSession({ headers: await headers() })
);

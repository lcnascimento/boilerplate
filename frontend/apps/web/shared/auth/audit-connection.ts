import { AsyncLocalStorage } from "node:async_hooks";
import { createHmac } from "node:crypto";

import { tryGetCurrentAuthEndpointContext } from "@better-auth/core/context";
import { getIP } from "better-auth/api";
import { CompiledQuery, Kysely, PostgresDialect, sql } from "kysely";

import type { ActorIds } from "./actor-ids";
import type { SignInFailureReason } from "./login-failures";
import type { BetterAuthOptions } from "better-auth";
import type { Pool } from "pg";

export interface ConnectionSettings {
  actorId: string;
  actorOrganizationId: string;
  ipHash: string;
}

interface RequestOrigin {
  userId: string | null;
  organizationId: string | null;
  headers: Headers | null;
}

const actorRequests = new AsyncLocalStorage<RequestOrigin>();

export function hashIp(key: string, ip: string): string {
  return createHmac("sha256", key).update(ip).digest("hex");
}

export function runAsActor<T>(
  actor: ActorIds,
  headers: Headers,
  run: () => Promise<T>
): Promise<T> {
  return actorRequests.run(
    { userId: actor.userId, organizationId: actor.organizationId, headers },
    run
  );
}

function endpointOrigin(): RequestOrigin | undefined {
  const endpoint = tryGetCurrentAuthEndpointContext();
  if (endpoint === undefined) return undefined;

  const session = endpoint.context.session;
  const activeOrganizationId: unknown = session?.session.activeOrganizationId;
  const headers = endpoint.request?.headers ?? endpoint.headers;
  return {
    userId: session?.user.id ?? null,
    organizationId:
      typeof activeOrganizationId === "string" ? activeOrganizationId : null,
    headers: headers === undefined ? null : new Headers(headers),
  };
}

export function currentConnectionSettings(
  ipHashKey: string,
  options: BetterAuthOptions
): ConnectionSettings {
  const origin = actorRequests.getStore() ?? endpointOrigin();
  const ip = origin?.headers == null ? null : getIP(origin.headers, options);
  return {
    actorId: origin?.userId ?? "",
    actorOrganizationId: origin?.organizationId ?? "",
    ipHash: ip === null ? "" : hashIp(ipHashKey, ip),
  };
}

const applySettings = `SELECT
  set_config('app.actor_id', $1, false),
  set_config('app.actor_organization_id', $2, false),
  set_config('app.ip_hash', $3, false)`;

export function createAuditedDatabase(
  pool: Pool,
  settings: () => ConnectionSettings
): Kysely<unknown> {
  return new Kysely({
    dialect: new PostgresDialect({
      pool,
      onReserveConnection: async (connection) => {
        const { actorId, actorOrganizationId, ipHash } = settings();
        await connection.executeQuery(
          CompiledQuery.raw(applySettings, [
            actorId,
            actorOrganizationId,
            ipHash,
          ])
        );
      },
    }),
  });
}

export async function recordSignInFailure(
  database: Kysely<unknown>,
  reason: SignInFailureReason
): Promise<void> {
  await sql`INSERT INTO identity_change (table_name, operation, fields, ip_hash)
            VALUES ('sign_in', 'FAILED', jsonb_build_object('reason', ${reason}::text),
                    NULLIF(current_setting('app.ip_hash', true), ''))`.execute(
    database
  );
}

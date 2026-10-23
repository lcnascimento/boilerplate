import { Pool } from "pg";

export function fixtureDatabase(): Pool {
  return new Pool({ connectionString: process.env.IAM_DATABASE_URL });
}

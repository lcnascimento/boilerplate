/**
 * Error returned by a backend service: go-kit's `APIError` body plus the HTTP
 * status. Network failures and responses outside the schema become synthetic
 * errors with the codes `UNAVAILABLE` and `UNEXPECTED`.
 */
export interface APIError {
  code: string;
  message: string;
  retryable: boolean;
  details?: { reasons?: string[] };
  status: number;
}

/** Outcome of every API call. Calls never throw: they return an error. */
export type Result<T> =
  { kind: "ok"; data: T } | { kind: "error"; error: APIError };

export interface APIError {
  code: string;
  message: string;
  retryable: boolean;
  details?: { reasons?: string[] };
  status: number;
}

export type Result<T> =
  { kind: "ok"; data: T } | { kind: "error"; error: APIError };

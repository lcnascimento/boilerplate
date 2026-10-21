import { useEffect } from "react";

import { reportBoundaryError } from "./faro";

export function useReportBoundaryError(
  error: Error & { digest?: string }
): void {
  useEffect(() => {
    reportBoundaryError(error);
  }, [error]);
}

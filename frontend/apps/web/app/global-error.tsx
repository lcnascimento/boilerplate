"use client";

import { useEffect, useSyncExternalStore } from "react";

import {
  defaultLocale,
  messages,
  parseLocale,
  type Locale,
} from "@boilerplate/i18n";
import { Button } from "@boilerplate/ui/components/button";

import { LOCALE_COOKIE_NAME } from "@/shared/cookies";
import { reportBoundaryError } from "@/shared/faro";

import "./globals.css";

interface GlobalErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

const subscribe = () => () => undefined;

function cookieLocale(): Locale {
  const entry = document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith(`${LOCALE_COOKIE_NAME}=`));
  return parseLocale(entry?.slice(LOCALE_COOKIE_NAME.length + 1));
}

// Next renders global-error in place of the root layout, outside every provider.
export default function GlobalError({ error, reset }: GlobalErrorProps) {
  const locale = useSyncExternalStore(
    subscribe,
    cookieLocale,
    () => defaultLocale
  );
  const { common, errors } = messages[locale];

  useEffect(() => {
    reportBoundaryError(error);
  }, [error]);

  return (
    <html lang={locale}>
      <body className="font-sans antialiased">
        <main className="flex min-h-svh flex-col items-center justify-center gap-2 p-4 text-center">
          <h1 className="text-2xl font-bold">{errors.generic}</h1>
          <p className="text-muted-foreground">{errors.retryHint}</p>
          {error.digest ? (
            <p className="text-xs text-muted-foreground">
              <code>{error.digest}</code>
            </p>
          ) : null}
          <Button className="mt-4" onClick={reset}>
            {common.tryAgain}
          </Button>
        </main>
      </body>
    </html>
  );
}

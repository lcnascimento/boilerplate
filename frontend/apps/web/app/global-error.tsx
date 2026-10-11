"use client";

import { useSyncExternalStore } from "react";

import {
  defaultLocale,
  messages,
  parseLocale,
  type Locale,
} from "@boilerplate/i18n";
import { Button } from "@boilerplate/ui/components/button";

import { LOCALE_COOKIE_NAME } from "@/shared/cookies";

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

// Replaces the root layout when it fails, so it renders outside every provider
// and loads the messages for the cookie locale itself (the default locale while
// prerendering, where there is no cookie).
export default function GlobalError({ error, reset }: GlobalErrorProps) {
  const locale = useSyncExternalStore(
    subscribe,
    cookieLocale,
    () => defaultLocale
  );
  const { common, errors } = messages[locale];

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

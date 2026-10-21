"use client";

import { ThemeProvider as NextThemesProvider } from "next-themes";

// The theme script only runs from the server HTML. When React renders the root layout on the client (a 404 or an
// error boundary), a JavaScript <script> makes it log a console.error; a data block does not.
const themeScriptProps =
  typeof window === "undefined" ? undefined : { type: "application/json" };

interface ThemeProviderProps {
  children: React.ReactNode;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  return (
    <NextThemesProvider
      disableTransitionOnChange
      enableSystem
      attribute="class"
      defaultTheme="system"
      scriptProps={themeScriptProps}
    >
      {children}
    </NextThemesProvider>
  );
}

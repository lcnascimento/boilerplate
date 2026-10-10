import { NextIntlClientProvider } from "next-intl";
import { getLocale } from "next-intl/server";

import { Toaster } from "@boilerplate/ui/components/sonner";
import { TooltipProvider } from "@boilerplate/ui/components/tooltip";

import { ThemeProvider } from "@/providers";

import type { Metadata } from "next";
import type { ReactNode } from "react";

import "./globals.css";

export const metadata: Metadata = {
  title: "Boilerplate",
};

interface RootLayoutProps {
  children: ReactNode;
}

export default async function RootLayout({ children }: RootLayoutProps) {
  const locale = await getLocale();

  return (
    // next-themes sets the theme class on <html> before hydration.
    <html suppressHydrationWarning lang={locale}>
      <body className="font-sans antialiased">
        <ThemeProvider>
          <NextIntlClientProvider>
            <TooltipProvider>
              {children}
              <Toaster />
            </TooltipProvider>
          </NextIntlClientProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}

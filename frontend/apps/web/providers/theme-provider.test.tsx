/** @vitest-environment jsdom */
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";

import { ThemeProvider } from "./theme-provider";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("renders on the client without a console error", () => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addListener: () => undefined,
    removeListener: () => undefined,
  }));
  const consoleError = vi.spyOn(console, "error");

  render(
    <ThemeProvider>
      <p>content</p>
    </ThemeProvider>
  );

  expect(screen.getByText("content")).toBeDefined();
  expect(consoleError).not.toHaveBeenCalled();
});

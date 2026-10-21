import { vi } from "vitest";

vi.mock("./core/logger", () => ({ logger: { error: vi.fn() } }));

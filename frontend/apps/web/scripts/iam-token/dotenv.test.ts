import { describe, expect, it } from "vitest";

import { withEnvValue } from "./dotenv";

describe("withEnvValue", () => {
  it("creates the file content when there is none", () => {
    expect(withEnvValue("", "IAM_TOKEN", "abc")).toBe("IAM_TOKEN=abc\n");
  });

  it("appends the key after the other lines", () => {
    expect(withEnvValue("OTHER=1", "IAM_TOKEN", "abc")).toBe(
      "OTHER=1\nIAM_TOKEN=abc\n"
    );
  });

  it("replaces the previous value and keeps the other lines", () => {
    expect(
      withEnvValue("OTHER=1\nIAM_TOKEN=old\nLAST=2\n", "IAM_TOKEN", "new")
    ).toBe("OTHER=1\nIAM_TOKEN=new\nLAST=2\n");
  });
});

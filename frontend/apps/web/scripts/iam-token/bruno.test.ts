import { readFileSync } from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { brunoCollections, IAM_TOKEN_AUTH } from "./bruno";

describe("Bruno collections", () => {
  it("authenticate every request with the IAM_TOKEN that iam:token writes", () => {
    const withoutTheToken = brunoCollections().filter(
      (dir) =>
        !readFileSync(path.join(dir, "opencollection.yml"), "utf8").includes(
          IAM_TOKEN_AUTH
        )
    );

    expect(withoutTheToken).toEqual([]);
  });
});

import { describe, expect, it } from "vitest";

import { signUpOrganizationName } from "./sign-up-body";

describe("signUpOrganizationName", () => {
  it("takes the trimmed Organization name from a sign-up request", () => {
    expect(
      signUpOrganizationName({ body: { organizationName: "  Acme  " } })
    ).toBe("Acme");
  });

  it.each([
    ["without an Organization name", { email: "ana@acme.test" }],
    ["with a blank Organization name", { organizationName: "   " }],
    [
      "with an Organization name over 100 characters",
      { organizationName: "a".repeat(101) },
    ],
    ["without a body", undefined],
  ])("refuses a sign-up request %s", (_, body) => {
    let refusal: unknown;
    try {
      signUpOrganizationName({ body });
    } catch (error) {
      refusal = error;
    }

    expect(refusal).toMatchObject({
      statusCode: 400,
      body: { code: "INVALID_ORGANIZATION_NAME" },
    });
  });

  it("owes no Organization for a User created outside any endpoint, like the seed's", () => {
    expect(signUpOrganizationName(null)).toBeNull();
  });
});

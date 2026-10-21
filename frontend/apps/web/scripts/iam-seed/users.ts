import type { MemberRole } from "@/shared/auth/members";

export const SEED_PASSWORD = "dev-password";

export interface SeedUser {
  name: string;
  email: string;
  role: MemberRole;
}

export interface SeedOrganization {
  name: string;
  slug: string;
  owner: SeedUser & { role: "owner" };
  members: readonly SeedUser[];
}

export const ACME_OWNER = {
  name: "Ana Owner",
  email: "owner@acme.test",
  role: "owner",
} as const satisfies SeedUser;

export const MULTI_ORGANIZATION_ADMIN = {
  name: "Eva Admin",
  email: "admin@multi.test",
  role: "admin",
} as const satisfies SeedUser;

export const seedOrganizations: readonly SeedOrganization[] = [
  {
    name: "Acme",
    slug: "acme",
    owner: ACME_OWNER,
    members: [
      { name: "Bruno Admin", email: "admin@acme.test", role: "admin" },
      { name: "Carla Viewer", email: "viewer@acme.test", role: "viewer" },
      MULTI_ORGANIZATION_ADMIN,
    ],
  },
  {
    name: "Globex",
    slug: "globex",
    owner: { name: "Diego Owner", email: "owner@globex.test", role: "owner" },
    members: [MULTI_ORGANIZATION_ADMIN],
  },
];

export const seedUsers: readonly SeedUser[] = [
  ...new Map(
    seedOrganizations
      .flatMap(({ owner, members }) => [owner, ...members])
      .map((user) => [user.email, user])
  ).values(),
];

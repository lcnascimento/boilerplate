export const SEED_PASSWORD = "dev-password";

export interface SeedUser {
  name: string;
  email: string;
  role: "owner" | "admin" | "viewer";
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

export const seedOrganizations: readonly SeedOrganization[] = [
  {
    name: "Acme",
    slug: "acme",
    owner: ACME_OWNER,
    members: [
      { name: "Bruno Admin", email: "admin@acme.test", role: "admin" },
      { name: "Carla Viewer", email: "viewer@acme.test", role: "viewer" },
    ],
  },
  {
    name: "Globex",
    slug: "globex",
    owner: { name: "Diego Owner", email: "owner@globex.test", role: "owner" },
    members: [],
  },
];

export const seedUsers: readonly SeedUser[] = seedOrganizations.flatMap(
  ({ owner, members }) => [owner, ...members]
);

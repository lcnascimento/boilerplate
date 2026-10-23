import { getIam } from "@/shared/auth/auth";

import { createUserOutsideSignUp } from "./create-user";
import { SEED_PASSWORD, seedOrganizations, type SeedUser } from "./users";

const { auth, database } = getIam();
const context = await auth.$context;

async function ensureUser({ name, email }: SeedUser): Promise<string> {
  const existing = await context.internalAdapter.findUserByEmail(email);
  if (existing) return existing.user.id;

  const userId = await createUserOutsideSignUp(context, {
    name,
    email,
    password: SEED_PASSWORD,
  });
  console.log(`created user ${email}`);
  return userId;
}

async function ensureOrganization(
  name: string,
  slug: string,
  ownerId: string
): Promise<string> {
  const existing = await context.adapter.findOne<{ id: string }>({
    model: "organization",
    where: [{ field: "slug", value: slug }],
  });
  if (existing) return existing.id;

  const created = await auth.api.createOrganization({
    body: { name, slug, userId: ownerId },
  });
  console.log(`created organization ${slug}`);
  return created.id;
}

async function ensureMember(
  organizationId: string,
  userId: string,
  { email, role }: SeedUser
): Promise<void> {
  const existing = await context.adapter.findOne({
    model: "member",
    where: [
      { field: "organizationId", value: organizationId },
      { field: "userId", value: userId },
    ],
  });
  if (existing) return;

  await auth.api.addMember({ body: { organizationId, userId, role } });
  console.log(`added ${email} as ${role}`);
}

try {
  for (const { name, slug, owner, members } of seedOrganizations) {
    const organizationId = await ensureOrganization(
      name,
      slug,
      await ensureUser(owner)
    );
    for (const member of members) {
      await ensureMember(organizationId, await ensureUser(member), member);
    }
  }
  console.log("iam seed is up to date");
} finally {
  await database.destroy();
}

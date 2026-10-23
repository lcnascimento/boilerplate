import type { getIam } from "@/shared/auth/auth";

type AuthContext = Awaited<ReturnType<typeof getIam>["auth"]["$context"]>;

export interface NewUser {
  name: string;
  email: string;
  password: string;
}

export async function createUserOutsideSignUp(
  context: AuthContext,
  { name, email, password }: NewUser
): Promise<string> {
  const user = await context.internalAdapter.createUser(
    { name, email },
    { method: "seed" }
  );
  await context.internalAdapter.linkAccount({
    userId: user.id,
    providerId: "credential",
    accountId: user.id,
    password: await context.password.hash(password),
  });
  return user.id;
}

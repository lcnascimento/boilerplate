export interface Organization {
  id: string;
  name: string;
}

export interface UserOrganizations {
  active: Organization;
  organizations: readonly Organization[];
}

// `organizations` is oldest membership first, the same order session.create.before picks the first active one from.
export function pickActiveOrganization(
  organizations: readonly Organization[],
  activeOrganizationId: string | null
): Organization | null {
  return (
    organizations.find(({ id }) => id === activeOrganizationId) ??
    organizations[0] ??
    null
  );
}

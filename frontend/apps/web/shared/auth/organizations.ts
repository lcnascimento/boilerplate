export interface Organization {
  id: string;
  name: string;
}

export interface UserOrganizations {
  active: Organization;
  organizations: readonly Organization[];
}

export function pickActiveOrganization(
  organizationsOldestMembershipFirst: readonly Organization[],
  activeOrganizationId: string | null
): Organization | null {
  return (
    organizationsOldestMembershipFirst.find(
      ({ id }) => id === activeOrganizationId
    ) ??
    organizationsOldestMembershipFirst[0] ??
    null
  );
}

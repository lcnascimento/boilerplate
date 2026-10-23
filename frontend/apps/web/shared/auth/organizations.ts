export interface Organization {
  id: string;
  name: string;
}

export interface UserOrganizations {
  active: Organization;
  organizations: readonly Organization[];
}

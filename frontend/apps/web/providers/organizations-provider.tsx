"use client";

import { createContext, useContext, type ReactNode } from "react";

import type { UserOrganizations } from "@/shared/auth/organizations";

const OrganizationsContext = createContext<UserOrganizations | null>(null);

interface OrganizationsProviderProps {
  organizations: UserOrganizations;
  children: ReactNode;
}

export function OrganizationsProvider({
  organizations,
  children,
}: OrganizationsProviderProps) {
  return (
    <OrganizationsContext value={organizations}>
      {children}
    </OrganizationsContext>
  );
}

export function useOrganizations(): UserOrganizations {
  const organizations = useContext(OrganizationsContext);
  if (organizations === null) {
    throw new Error(
      "useOrganizations must be used within OrganizationsProvider"
    );
  }
  return organizations;
}

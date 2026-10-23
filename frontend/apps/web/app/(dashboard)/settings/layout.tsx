import { SettingsNav } from "./_components";

import type { ReactNode } from "react";

interface SettingsLayoutProps {
  children: ReactNode;
}

export default function SettingsLayout({ children }: SettingsLayoutProps) {
  return (
    <div className="grid gap-6 md:grid-cols-[12rem_minmax(0,1fr)]">
      <SettingsNav />
      <div className="grid content-start gap-4">{children}</div>
    </div>
  );
}

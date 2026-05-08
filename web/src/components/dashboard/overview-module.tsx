"use client";

import type { ReactNode } from "react";

export interface DashboardNavLink {
  id: string;
  label: string;
  icon: ReactNode;
}

export function OverviewModule({ metrics }: { metrics: ReactNode }) {
  return (
    <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {metrics}
    </section>
  );
}

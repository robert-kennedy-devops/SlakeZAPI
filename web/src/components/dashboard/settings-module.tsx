"use client";

import type { ReactNode } from "react";

export function SettingsModule({
  billingManagement,
  accountSecurity,
  workspaceTools,
}: {
  billingManagement: ReactNode;
  accountSecurity: ReactNode;
  workspaceTools: ReactNode;
}) {
  return (
    <>
      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]" id="billing">
        {billingManagement}
      </section>
      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]" id="identity">
        {accountSecurity}
      </section>
      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]" id="tools">
        {workspaceTools}
      </section>
    </>
  );
}

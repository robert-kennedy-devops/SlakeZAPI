"use client";

import type { ReactNode } from "react";

export function AutomationModule({
  advancedMessaging,
  groupManagement,
}: {
  advancedMessaging: ReactNode;
  groupManagement: ReactNode;
}) {
  return (
    <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]" id="advanced">
      <div className="grid gap-6">{advancedMessaging}</div>
      <div className="grid gap-6">{groupManagement}</div>
    </section>
  );
}
